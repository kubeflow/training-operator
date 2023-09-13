# Copyright 2023 The Kubeflow Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import multiprocessing
import logging
import time
from typing import Optional, Callable, List, Dict, Any, Set
import queue
from kubernetes import client, config, watch

from kubeflow.training import models
from kubeflow.training.api_client import ApiClient
from kubeflow.training.constants import constants
from kubeflow.training.utils import utils

logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)

status_logger = utils.StatusLogger(
    header="{:<30.30} {:<20.20} {}".format("NAME", "STATE", "TIME"),
    column_format="{:<30.30} {:<20.20} {}",
)


class TrainingClient(object):
    def __init__(
        self,
        config_file: Optional[str] = None,
        context: Optional[str] = None,
        client_configuration: Optional[client.Configuration] = None,
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.PYTORCHJOB_KIND,
    ):
        """TrainingClient constructor.

        Args:
            config_file: Path to the kube-config file. Defaults to ~/.kube/config.
            context: Set the active context. Defaults to current_context from the kube-config.
            client_configuration: Client configuration for cluster authentication.
                You have to provide valid configuration with Bearer token or
                with username and password. You can find an example here:
                https://github.com/kubernetes-client/python/blob/67f9c7a97081b4526470cad53576bc3b71fa6fcc/examples/remote_cluster.py#L31
            namespace: Target Kubernetes namespace. By default it takes namespace
                from `/var/run/secrets/kubernetes.io/serviceaccount/namespace` location
                or set as `default`. Namespace can be overridden during method invocations.
            job_kind: Target Training Job kind (e.g. `TFJob`, `PyTorchJob`, `MPIJob`).
                Job kind can be overridden during method invocations.
                The default Job kind is `PyTorchJob`.

        Raises:
            ValueError: Job kind is invalid.
        """

        # If client configuration is not set, use kube-config to access Kubernetes APIs.
        if client_configuration is None:
            # Load kube-config or in-cluster config.
            if config_file or not utils.is_running_in_k8s():
                config.load_kube_config(config_file=config_file, context=context)
            else:
                config.load_incluster_config()

        k8s_client = client.ApiClient(client_configuration)
        self.custom_api = client.CustomObjectsApi(k8s_client)
        self.core_api = client.CoreV1Api(k8s_client)
        self.api_client = ApiClient()

        self.namespace = namespace
        if job_kind not in constants.JOB_PARAMETERS:
            raise ValueError(
                f"Job kind must be one of these: {list(constants.JOB_PARAMETERS.keys())}"
            )
        self.job_kind = job_kind

    def create_job(
        self,
        job: Optional[constants.JOB_MODELS_TYPE] = None,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        base_image: Optional[str] = None,
        train_func: Optional[Callable] = None,
        parameters: Optional[Dict[str, Any]] = None,
        num_worker_replicas: Optional[int] = None,
        num_chief_replicas: Optional[int] = None,
        num_ps_replicas: Optional[int] = None,
        packages_to_install: Optional[List[str]] = None,
        pip_index_url: str = constants.DEFAULT_PIP_INDEX_URL,
    ):
        """Create the Training Job.
        Job can be created using one of the following options:

        - Define custom resource object in `job` parameter (e.g. TFJob or PyTorchJob).
        - Define training function in `train_func` parameter and number of workers.
        - Define Docker image in `base_image` parameter and number of workers.

        Args:
            job: Job object. Object must be one of these types: KubeflowOrgV1TFJob,
                KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob, etc.
            name: Name for the Job. It must be set if `job` parameter is omitted.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). It must be set if
                `job` parameter is omitted. By default Job kind is taken from
                `TrainingClient` object.
            base_image: Image that Job uses to train the model on each training replica.
                If `train_func` parameter is set, this image is used to execute the training
                function. The `constants` module contains some base images, the default image
                is `docker.io/pytorch/pytorch:1.12.1-cuda11.3-cudnn8-runtime`
            train_func: Function that Job uses to train the model on each training replica.
                This function must be Callable. Optionally, this function might have one dict
                argument to define input parameters for the function. If `train_func` is
                set, Base Image must support `bash` CLI to execute the training script.
            parameters: Dict of input parameters that training function might receive.
            num_worker_replicas: Number of Worker replicas for the Job.
            num_chief_replicas: Number of Chief replicas for the TFJob. Number
                of Chief replicas can't be more than 1.
            num_ps_replicas: Number of Parameter Server replicas for the TFJob.
            packages_to_install: List of Python packages to install in addition
                to the base image packages if `train_func` parameter is set.
                These packages are installed before executing the objective function.
            pip_index_url: The PyPI url from which to install Python packages.

        Raises:
            ValueError: Invalid input parameters.
            TimeoutError: Timeout to create Job.
            RuntimeError: Failed to create Job.
        """

        # When Job is set, only namespace arg is allowed.
        if job is not None:
            for key, value in locals().items():
                if (
                    key not in ["self", "job", "namespace", "pip_index_url"]
                    and value is not None
                ):
                    raise ValueError(
                        "If `job` is set only `namespace` argument is allowed. "
                        f"Argument `{key}` must be None."
                    )

        namespace = namespace or self.namespace
        job_kind = job_kind or self.job_kind
        if job is not None:
            job_kind = job.kind

        if job_kind not in constants.JOB_PARAMETERS:
            raise ValueError(
                f"Job kind must be one of these: {constants.JOB_PARAMETERS.keys()}"
            )

        # If Training function or base image is set, configure Job template.
        if train_func is not None or base_image is not None:
            # Job name must be set to configure Job template.
            if name is None:
                raise ValueError(
                    "Job name must be set to configure Job from function or image"
                )

            # Get Pod template spec from function or image.
            pod_template_spec = utils.get_pod_template_spec(
                job_kind=job_kind,
                base_image=base_image,
                train_func=train_func,
                parameters=parameters,
                packages_to_install=packages_to_install,
                pip_index_url=pip_index_url,
            )

            # Configure template for different Jobs.
            # TODO (andreyvelich): Add support for other kinds (e.g. MPIJob).
            if job_kind == constants.TFJOB_KIND:
                job = utils.get_tfjob_template(
                    name=name,
                    namespace=namespace,
                    pod_template_spec=pod_template_spec,
                    num_worker_replicas=num_worker_replicas,
                    num_chief_replicas=num_chief_replicas,
                    num_ps_replicas=num_ps_replicas,
                )
            elif job_kind == constants.PYTORCHJOB_KIND:
                job = utils.get_pytorchjob_template(
                    name=name,
                    namespace=namespace,
                    pod_template_spec=pod_template_spec,
                    num_worker_replicas=num_worker_replicas,
                )
            else:
                raise ValueError(
                    f"Job kind {job_kind} can't be created using function or image"
                )

        # Verify Job object type.
        if not isinstance(job, constants.JOB_MODELS):
            raise ValueError(f"Job must be one of these types: {constants.JOB_MODELS}")

        # Create the Training Job.
        try:
            self.custom_api.create_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                namespace,
                constants.JOB_PARAMETERS[job.kind]["plural"],
                job,
            )
        except multiprocessing.TimeoutError:
            raise TimeoutError(
                f"Timeout to create {job_kind}: {namespace}/{job.metadata.name}"
            )
        except Exception:
            raise RuntimeError(
                f"Failed to create {job_kind}: {namespace}/{job.metadata.name}"
            )

        logging.info(f"{job_kind} {namespace}/{job.metadata.name} has been created")

    def get_job(
        self,
        name: str,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> constants.JOB_MODELS_TYPE:
        """Get the Training Job.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            object: Job object. For example: KubeflowOrgV1PyTorchJob

        Raises:
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        namespace = namespace or self.namespace
        job_kind = job_kind or self.job_kind

        if job_kind not in constants.JOB_PARAMETERS:
            raise ValueError(
                f"Job kind must be one of these: {constants.JOB_PARAMETERS.keys()}"
            )

        try:
            thread = self.custom_api.get_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                namespace,
                constants.JOB_PARAMETERS[job_kind]["plural"],
                name,
                async_req=True,
            )
            response = utils.FakeResponse(thread.get(timeout))
            job = self.api_client.deserialize(
                response, constants.JOB_PARAMETERS[job_kind]["model"]
            )

        except multiprocessing.TimeoutError:
            raise TimeoutError(f"Timeout to get {job_kind}: {namespace}/{name}")
        except Exception:
            raise RuntimeError(f"Failed to get {job_kind}: {namespace}/{name}")

        return job

    def list_jobs(
        self,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> List[constants.JOB_MODELS_TYPE]:
        """List of all Training Jobs with specific kind in namespace.

        Args:
            namespace: Namespace to list the Jobs. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            list[object]: List of Job objects.
                For example: list of KubeflowOrgV1PyTorchJob objects. It returns empty list
                if Jobs can't be found.

        Raises:
            TimeoutError: Timeout to list Jobs
            RuntimeError: Failed to list Jobs
        """

        namespace = namespace or self.namespace
        job_kind = job_kind or self.job_kind

        if job_kind not in constants.JOB_PARAMETERS:
            raise ValueError(
                f"Job kind must be one of these: {constants.JOB_PARAMETERS.keys()}"
            )

        result = []
        try:
            thread = self.custom_api.list_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                namespace,
                constants.JOB_PARAMETERS[job_kind]["plural"],
                async_req=True,
            )
            response = thread.get(timeout)
            result = [
                self.api_client.deserialize(
                    utils.FakeResponse(item),
                    constants.JOB_PARAMETERS[job_kind]["model"],
                )
                for item in response.get("items")
            ]
        except multiprocessing.TimeoutError:
            raise TimeoutError(f"Timeout to list {job_kind}s in namespace: {namespace}")
        except Exception:
            raise RuntimeError(f"Failed to list {job_kind}s in namespace: {namespace}")

        return result

    def get_job_conditions(
        self,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        job: Optional[constants.JOB_MODELS_TYPE] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> List[models.V1JobCondition]:
        """Get the Training Job conditions. Training Job is in the condition when
        `status=True` for the appropriate condition `type`. For example,
        Training Job is Succeeded when `status=True` and `type=Succeeded`.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            job: Job object can be set to get the conditions. Object must be one of
                these types: KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob, etc.
                If this parameter is omitted, it gets Job with the given name and kind.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            list[V1JobCondition]: List of Job conditions with
                last transition time, last update time, message, reason, type, and
                status. It returns empty list if Job does not have any
                conditions yet.

        Raises:
            ValueError: Invalid input parameters.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        namespace = namespace or self.namespace
        job_kind = job_kind or self.job_kind

        if job_kind not in constants.JOB_PARAMETERS:
            raise ValueError(
                f"Job kind must be one of these: {constants.JOB_PARAMETERS.keys()}"
            )

        if job is not None and not isinstance(job, constants.JOB_MODELS):
            raise ValueError(f"Job must be one of these types: {constants.JOB_MODELS}")

        # If Job is not set, get the Training Job.
        if job is None:
            # Job name must be set when Job object is not set.
            if name is None:
                raise ValueError(
                    "Job name must be set to configure Job from function or image"
                )

            job = self.get_job(
                name=name,
                namespace=namespace,
                job_kind=job_kind,
                timeout=timeout,
            )
        if job.status and job.status.conditions and len(job.status.conditions) > 0:
            return job.status.conditions
        return []

    def is_job_created(
        self,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        job: Optional[constants.JOB_MODELS_TYPE] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> bool:
        """Check if Training Job is Created.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            job: Job object can be set to get the conditions. Object must be one of
                these types: KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob, etc.
                If this parameter is omitted, it gets Job with the given name and kind.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            bool: True if Job is Created, else False.

        Raises:
            ValueError: Invalid input parameters.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_CREATED,
        )

    def is_job_running(
        self,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        job: Optional[constants.JOB_MODELS_TYPE] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> bool:
        """Check if Training Job is Running.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            job: Job object can be set to get the conditions. Object must be one of
                these types: KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob, etc.
                If this parameter is omitted, it gets Job with the given name and kind.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            bool: True if Job is Running, else False.

        Raises:
            ValueError: Invalid input parameters.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_RUNNING,
        )

    def is_job_restarting(
        self,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        job: Optional[constants.JOB_MODELS_TYPE] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> bool:
        """Check if Training Job is Restarting.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            job: Job object can be set to get the conditions. Object must be one of
                these types: KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob, etc.
                If this parameter is omitted, it gets Job with the given name and kind.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            bool: True if Job is Restarting, else False.

        Raises:
            ValueError: Invalid input parameters.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_RESTARTING,
        )

    def is_job_succeeded(
        self,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        job: Optional[constants.JOB_MODELS_TYPE] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> bool:
        """Check if Training Job is Succeeded.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            job: Job object can be set to get the conditions. Object must be one of
                these types: KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob, etc.
                If this parameter is omitted, it gets Job with the given name and kind.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            bool: True if Job is Succeeded, else False.

        Raises:
            ValueError: Invalid input parameters.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_SUCCEEDED,
        )

    def is_job_failed(
        self,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        job: Optional[constants.JOB_MODELS_TYPE] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> bool:
        """Check if Training Job is Failed.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            job: Job object can be set to get the conditions. Object must be one of
                these types: KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob, etc.
                If this parameter is omitted, it gets Job with the given name and kind.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            bool: True if Job is Failed, else False.

        Raises:
            ValueError: Invalid input parameters.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_FAILED,
        )

    def wait_for_job_conditions(
        self,
        name: str,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        expected_conditions: Set = {constants.JOB_CONDITION_SUCCEEDED},
        wait_timeout: int = 600,
        polling_interval: int = 15,
        callback: Optional[Callable] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> constants.JOB_MODELS_TYPE:
        """Wait until Training Job reaches any of the specified conditions.
        By default it waits for the Succeeded condition.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            expected_conditions: Set of expected conditions. It must be subset of this:
                `{"Created", "Running", "Restarting", "Succeeded", "Failed"}`
            wait_timeout: How many seconds to wait until Job reaches one of
                the expected conditions.
            polling_interval: The polling interval in seconds to get Job status.
            callback: Callback function that is invoked after Job
                status is polled. This function takes a single argument which
                is current Job object.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            object: Job object. For example: KubeflowOrgV1PyTorchJob

        Raises:
            ValueError: Invalid input parameters.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job or Job reaches unexpected Failed condition.
        """

        namespace = namespace or self.namespace
        job_kind = job_kind or self.job_kind

        if not expected_conditions.issubset(constants.JOB_CONDITIONS):
            raise ValueError(
                f"Expected conditions: {expected_conditions} must be subset of \
                    {constants.JOB_CONDITIONS}"
            )
        for _ in range(round(wait_timeout / polling_interval)):
            # We should get Job only once per cycle and check the statuses.
            job = self.get_job(
                name=name,
                namespace=namespace,
                job_kind=job_kind,
                timeout=timeout,
            )

            # Get Job conditions.
            conditions = self.get_job_conditions(job=job, timeout=timeout)
            if len(conditions) > 0:
                status_logger(
                    name,
                    conditions[-1].type,
                    conditions[-1].last_transition_time,
                )

            # Execute callback function is it is set.
            if callback:
                callback(job)

            # Raise an exception if Job is Failed and Failed is not expected condition.
            if (
                constants.JOB_CONDITION_FAILED not in conditions
                and utils.has_condition(conditions, constants.JOB_CONDITION_FAILED)
            ):
                raise RuntimeError(
                    f"{job_kind} {namespace}/{name} is Failed. "
                    f"{job_kind} conditions: {job.status.conditions}"
                )

            # Return Job when it reaches expected condition.
            for expected_condition in expected_conditions:
                if utils.has_condition(conditions, expected_condition):
                    return job

            time.sleep(polling_interval)

        raise TimeoutError(
            f"Timeout waiting for {job_kind}: {namespace}/{name} to reach expected conditions: \
                {expected_conditions}"
        )

    def get_job_pod_names(
        self,
        name: str,
        namespace: Optional[str] = None,
        is_master: bool = False,
        replica_type: Optional[str] = None,
        replica_index: Optional[int] = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> List[str]:
        """Get pod names for the Training Job.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            is_master: Whether to get pods only with the label
                `training.kubeflow.org/job-role: master`.
            replica_type: Type of the Job replica.
                For TFJob one of `Chief`, `PS`, or `worker`.

                For PyTorchJob one of `master` or `worker`.

                For MXJob one of `scheduler`, `server`, or `worker`.

                For XGBoostJob one of `master` or `worker`.

                For MPIJob one of `launcher` or `worker`.

                For PaddleJob one of `master` or `worker`.

            replica_index: Index for the Job replica.
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            list[str]: List of the Job pod names.

        Raises:
            ValueError: Job replica type is invalid.
            TimeoutError: Timeout to get Job pods.
            RuntimeError: Failed to get Job pods.
        """

        namespace = namespace or self.namespace

        if (
            replica_type is not None
            and replica_type not in constants.TFJOB_REPLICA_TYPES
            and replica_type not in constants.PYTORCHJOB_REPLICA_TYPES
            and replica_type not in constants.MXJOB_REPLICA_TYPES
            and replica_type not in constants.XGBOOSTJOB_REPLICA_TYPES
            and replica_type not in constants.MPIJOB_REPLICA_TYPES
            and replica_type not in constants.PADDLEJOB_REPLICA_TYPES
        ):
            raise ValueError(
                f"TFJob replica type must be one of {constants.TFJOB_REPLICA_TYPES}\n"
                f"PyTorchJob replica type must be one of {constants.PYTORCHJOB_REPLICA_TYPES}\n"
                f"MXJob replica type must be one of {constants.MXJOB_REPLICA_TYPES}\n"
                f"XGBoostJob replica type must be one of {constants.XGBOOSTJOB_REPLICA_TYPES}\n"
                f"MPIJob replica type must be one of {constants.MPIJOB_REPLICA_TYPES}\n"
                f"PaddleJob replica type must be one of {constants.PADDLEJOB_REPLICA_TYPES}"
            )

        label_selector = f"{constants.JOB_NAME_LABEL}={name}"

        # Add Job role label if that is required.
        if is_master:
            label_selector += f",{constants.JOB_ROLE_LABEL}={constants.JOB_ROLE_MASTER}"

        # Add Replica type label if that is required.
        if replica_type:
            label_selector += (
                f",{constants.REPLICA_TYPE_LABEL}={str.lower(replica_type)}"
            )

        # Add Replica index label if that is required.
        if replica_index is not None:
            label_selector += f",{constants.REPLICA_INDEX_LABEL}={replica_index}"

        # List Training Job pods.
        pods = []
        try:
            thread = self.core_api.list_namespaced_pod(
                namespace,
                label_selector=label_selector,
                async_req=True,
            )
            response = thread.get(timeout)
        except multiprocessing.TimeoutError:
            raise TimeoutError(f"Timeout to list pods for Job: {namespace}/{name}")
        except Exception:
            raise RuntimeError(f"Failed to list pods for Job: {namespace}/{name}")

        for pod in response.items:
            pods.append(pod.metadata.name)
        return pods

    def get_job_logs(
        self,
        name: str,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        is_master: bool = True,
        replica_type: Optional[str] = None,
        replica_index: Optional[int] = None,
        follow: bool = False,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Print the training logs for the Job. By default it returns logs from
        the `master` pod.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            is_master: Whether to get logs for the pod with the label
                `training.kubeflow.org/job-role: master`.
            replica_type: Optional, type of the Job replica.
                For TFJob one of `chief`, `ps`, or `worker`.

                For PyTorchJob one of `master` or `worker`.

                For MXJob one of `scheduler`, `server`, or `worker`.

                For XGBoostJob one of `master` or `worker`.

                For MPIJob one of `launcher` or `worker`.

                For PaddleJob one of `master` or `worker`.
            replica_index: Optional, index for the Job replica.
            container: Pod container to get the logs.
            follow: Whether to follow the log stream of the pod.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Raises:
            ValueError: Job replica type is invalid.
            TimeoutError: Timeout to get Job pods.
            RuntimeError: Failed to get Job pods.
        """

        namespace = namespace or self.namespace
        job_kind = job_kind or self.job_kind

        pods = self.get_job_pod_names(
            name=name,
            namespace=namespace,
            is_master=is_master,
            replica_type=replica_type,
            replica_index=replica_index,
            timeout=timeout,
        )

        if pods and follow:
            log_streams = []
            for pod in pods:
                log_streams.append(
                    watch.Watch().stream(
                        self.core_api.read_namespaced_pod_log,
                        name=pod,
                        namespace=namespace,
                        container=constants.JOB_PARAMETERS[job_kind]["container"],
                    )
                )
            finished = [False for _ in log_streams]

            # Create thread and queue per stream, for non-blocking iteration
            log_queue_pool = utils.get_log_queue_pool(log_streams)

            # Iterate over every watching pods' log queue
            while True:
                for index, log_queue in enumerate(log_queue_pool):
                    if all(finished):
                        return
                    if finished[index]:
                        continue
                    # grouping the every 50 log lines of the same pod
                    for _ in range(50):
                        try:
                            logline = log_queue.get(timeout=1)
                            if logline is None:
                                finished[index] = True
                                break
                            logging.info("[Pod %s]: %s", pods[index], logline)
                        except queue.Empty:
                            break
        elif pods:
            for pod in pods:
                try:
                    pod_logs = self.core_api.read_namespaced_pod_log(
                        pod,
                        namespace,
                        container=constants.JOB_PARAMETERS[job_kind]["container"],
                    )
                    logging.info("The logs of pod %s:\n %s", pod, pod_logs)
                except Exception:
                    raise RuntimeError(f"Failed to read logs for pod {namespace}/{pod}")

    def update_job(
        self,
        job: constants.JOB_MODELS_TYPE,
        name: str,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
    ):
        """Update the Training Job by using patch Kubernetes API.

        Args:
            job: Job object. For example, object with type
                KubeflowOrgV1TFJob or KubeflowOrgV1PyTorchJob.
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
        Raises:
            TimeoutError: Timeout to update Job
            RuntimeError: Failed to update Job
        """

        namespace = namespace or self.namespace
        job_kind = job_kind or self.job_kind

        if job_kind not in constants.JOB_PARAMETERS:
            raise ValueError(
                f"Job kind must be one of these: {constants.JOB_PARAMETERS.keys()}"
            )

        try:
            self.custom_api.patch_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                namespace,
                constants.JOB_PARAMETERS[job_kind]["plural"],
                name,
                job,
            )
        except multiprocessing.TimeoutError:
            raise TimeoutError(f"Timeout to update {job_kind}: {namespace}/{name}")
        except Exception:
            raise RuntimeError(f"Failed to update {job_kind}: {namespace}/{name}")

        logging.info(f"{job_kind} {namespace}/{name} has been updated")

    def delete_job(
        self,
        name: str,
        namespace: Optional[str] = None,
        job_kind: Optional[str] = None,
        delete_options: Optional[client.V1DeleteOptions] = None,
    ):
        """Delete the Training Job

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job. By default namespace is taken from
                `TrainingClient` object.
            job_kind: Kind for the Job (e.g. `TFJob` or `PyTorchJob`). By default Job kind
                is taken from `TrainingClient` object.
            delete_options: Optional, V1DeleteOptions to set while deleting
                the Job. For example, grace period seconds.

        Raises:
            TimeoutError: Timeout to delete Job.
            RuntimeError: Failed to delete Job.
        """

        namespace = namespace or self.namespace
        job_kind = job_kind or self.job_kind

        try:
            self.custom_api.delete_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                namespace,
                constants.JOB_PARAMETERS[job_kind]["plural"],
                name=name,
                body=delete_options,
            )
        except multiprocessing.TimeoutError:
            raise TimeoutError(f"Timeout to delete {job_kind}: {namespace}/{name}")
        except Exception:
            raise RuntimeError(f"Failed to delete {job_kind}: {namespace}/{name}")

        logging.info(f"{job_kind} {namespace}/{name} has been deleted")
