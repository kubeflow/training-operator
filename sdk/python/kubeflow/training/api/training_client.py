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
from typing import Callable, List, Dict, Any, Set
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
        config_file: str = None,
        context: str = None,
        client_configuration: client.Configuration = None,
    ):
        """TrainingClient constructor.

        Args:
            config_file: Path to the kube-config file. Defaults to ~/.kube/config.
            context: Set the active context. Defaults to current_context from the kube-config.
            client_configuration: Client configuration for cluster authentication.
                You have to provide valid configuration with Bearer token or
                with username and password.
                You can find an example here: https://github.com/kubernetes-client/python/blob/67f9c7a97081b4526470cad53576bc3b71fa6fcc/examples/remote_cluster.py#L31
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

    # ------------------------------------------------------------------------ #
    # Common Training Client APIs.
    # ------------------------------------------------------------------------ #
    def get_job_conditions(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.TFJOB_KIND,
        job: object = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Get the Training Job conditions. Training Job is in the condition when
        `status=True` for the appropriate condition `type`. For example,
        Training Job is Succeeded when `status=True` and `type=Succeeded`.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
            job_kind: Kind for the Training job to get conditions.
                It should be one of these: `TFJob, PyTorchJob, MXJob, XGBoostJob, MPIJob, or PaddleJob`.
            job: Optionally, Training Job object can be set to get the conditions.
                It should be type of `KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob,
                KubeflowOrgV1XGBoostJob, KubeflowOrgV1MPIJob, or KubeflowOrgV1PaddleJob`
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            list[V1JobCondition]: List of Job conditions with
                last transition time, last update time, message, reason, type, and
                status. It returns empty list if Training Job does not have any
                conditions yet.

        Raises:
            ValueError: Job kind is invalid.
            TimeoutError: Timeout to get Training Job.
            RuntimeError: Failed to get Training Job.
        """

        models = tuple([d["model"] for d in list(constants.JOB_KINDS.values())])
        if job is not None and not isinstance(job, models):
            raise ValueError(f"Job must be one of these types: {models}")

        # If Job is not set, get the Training Job.
        if job is None:
            if job_kind not in constants.JOB_KINDS:
                raise ValueError(
                    f"Job kind must be one of these: {list(constants.JOB_KINDS.keys())}"
                )
            job = utils.get_job(
                custom_api=self.custom_api,
                api_client=self.api_client,
                name=name,
                namespace=namespace,
                job_model=constants.JOB_KINDS[job_kind]["model"],
                job_kind=job_kind,
                job_plural=constants.JOB_KINDS[job_kind]["plural"],
                timeout=timeout,
            )
        if job.status and job.status.conditions and len(job.status.conditions) > 0:
            return job.status.conditions
        return []

    def is_job_created(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.TFJOB_KIND,
        job: object = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Check if Training Job is Created.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
            job_kind: Kind for the Training job to check the status.
                It should be one of these: `TFJob, PyTorchJob, MXJob, XGBoostJob, MPIJob, or PaddleJob`.
            job: Optionally, Training Job object can be set to check the status.
                It should be type of `KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob,
                KubeflowOrgV1XGBoostJob, KubeflowOrgV1MPIJob, or KubeflowOrgV1PaddleJob`
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            bool: True if Job is Created, else False.

        Raises:
            ValueError: Job kind is invalid.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_CREATED,
        )

    def is_job_running(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.TFJOB_KIND,
        job: object = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Check if Training Job is Running.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
            job_kind: Kind for the Training job to check the status.
                It should be one of these: `TFJob, PyTorchJob, MXJob, XGBoostJob, MPIJob, or PaddleJob`.
            job: Optionally, Training Job object can be set to check the status.
                It should be type of `KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob,
                KubeflowOrgV1XGBoostJob, KubeflowOrgV1MPIJob, or KubeflowOrgV1PaddleJob`
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            bool: True if Job is Running, else False.

        Raises:
            ValueError: Job kind is invalid.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_RUNNING,
        )

    def is_job_restarting(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.TFJOB_KIND,
        job: object = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Check if Training Job is Restarting.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
            job_kind: Kind for the Training job to check the status.
                It should be one of these: `TFJob, PyTorchJob, MXJob, XGBoostJob, MPIJob, or PaddleJob`.
            job: Optionally, Training Job object can be set to check the status.
                It should be type of `KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob,
                KubeflowOrgV1XGBoostJob, KubeflowOrgV1MPIJob, or KubeflowOrgV1PaddleJob`
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            bool: True if Job is Restarting, else False.

        Raises:
            ValueError: Job kind is invalid.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_RESTARTING,
        )

    def is_job_succeeded(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.TFJOB_KIND,
        job: object = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Check if Training Job is Succeeded.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
            job_kind: Kind for the Training job to check the status.
                It should be one of these: `TFJob, PyTorchJob, MXJob, XGBoostJob, MPIJob, or PaddleJob`.
            job: Optionally, Training Job object can be set to check the status.
                It should be type of `KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob,
                KubeflowOrgV1XGBoostJob, KubeflowOrgV1MPIJob, or KubeflowOrgV1PaddleJob`
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            bool: True if Job is Succeeded, else False.

        Raises:
            ValueError: Job kind is invalid.
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        return utils.has_condition(
            self.get_job_conditions(name, namespace, job_kind, job, timeout),
            constants.JOB_CONDITION_SUCCEEDED,
        )

    def is_job_failed(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.TFJOB_KIND,
        job: object = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Check if Training Job is Failed.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
            job_kind: Kind for the Training job to check the status.
                It should be one of these: `TFJob, PyTorchJob, MXJob, XGBoostJob, MPIJob, or PaddleJob`.
            job: Optionally, Training Job object can be set to check the status.
                It should be type of `KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob, KubeflowOrgV1MXJob,
                KubeflowOrgV1XGBoostJob, KubeflowOrgV1MPIJob, or KubeflowOrgV1PaddleJob`
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            bool: True if Job is Failed, else False.

        Raises:
            ValueError: Job kind is invalid.
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
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.TFJOB_KIND,
        expected_conditions: Set = {constants.JOB_CONDITION_SUCCEEDED},
        timeout: int = 600,
        polling_interval: int = 15,
        callback: Callable = None,
        apiserver_timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Wait until Training Job reaches any of the specified conditions.
        By default it waits for the Succeeded condition.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
            job_kind: Kind for the Training job to wait for conditions.
                It should be one of these: `TFJob, PyTorchJob, MXJob, XGBoostJob, MPIJob, or PaddleJob`.
            expected_conditions: Set of expected conditions. It must be subset of this:
                `{"Created", "Running", "Restarting", "Succeeded", "Failed"}`
            timeout: How many seconds to wait until Job reaches one of
                the expected conditions.
            polling_interval: The polling interval in seconds to get Job status.
            callback: Optional callback function that is invoked after Job
                status is polled. This function takes a single argument which
                is current Job object.
            apiserver_timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            object: Training Job object of type `KubeflowOrgV1TFJob, KubeflowOrgV1PyTorchJob,
            KubeflowOrgV1MXJob, KubeflowOrgV1XGBoostJob, KubeflowOrgV1MPIJob, or
            KubeflowOrgV1PaddleJob` which is reached required condition.

        Raises:
            ValueError: Expected conditions are invalid or Job kind is invalid
            TimeoutError: Timeout to get Job.
            RuntimeError: Failed to get Job.
        """

        if not expected_conditions.issubset(constants.JOB_CONDITIONS):
            raise ValueError(
                f"Expected conditions: {expected_conditions} must be subset of {constants.JOB_CONDITIONS}"
            )
        for _ in range(round(timeout / polling_interval)):

            # We should get Job only once per cycle and check the statuses.
            job = utils.get_job(
                custom_api=self.custom_api,
                api_client=self.api_client,
                name=name,
                namespace=namespace,
                job_model=constants.JOB_KINDS[job_kind]["model"],
                job_kind=job_kind,
                job_plural=constants.JOB_KINDS[job_kind]["plural"],
                timeout=apiserver_timeout,
            )
            conditions = self.get_job_conditions(
                name, namespace, job_kind, job, timeout
            )
            if len(conditions) > 0:
                status_logger(
                    name, conditions[-1].type, conditions[-1].last_transition_time,
                )
            # Execute callback function.
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
            f"Timeout waiting for {job_kind}: {namespace}/{name} to reach expected conditions: {expected_conditions}"
        )

    def get_job_pod_names(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        is_master: bool = False,
        replica_type: str = None,
        replica_index: int = None,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Get pod names for the Training Job.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
            is_master: Whether to get pods only with the label
                `training.kubeflow.org/job-role: master`.
            replica_type: Optional, type of the Job replica.
                For TFJob one of `chief`, `ps`, or `worker`.

                For PyTorchJob one of `master` or `worker`.

                For MXJob one of `scheduler`, `server`, or `worker`.

                For XGBoostJob one of `master` or `worker`.

                For MPIJob one of `launcher` or `worker`.

                For PaddleJob one of `master` or `worker`.

            replica_index: Optional, index for the Job replica.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            list[str]: List of the Job pod names.

        Raises:
            ValueError: Job replica type is invalid.
            TimeoutError: Timeout to get Job pods.
            RuntimeError: Failed to get Job pods.
        """

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
                namespace, label_selector=label_selector, async_req=True,
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
        namespace: str = utils.get_default_target_namespace(),
        is_master: bool = True,
        replica_type: str = None,
        replica_index: int = None,
        container: str = constants.TFJOB_CONTAINER,
        follow: bool = False,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Print the training logs for the Job. By default it returns logs from
        the `master` pod.

        Args:
            name: Name for the Job.
            namespace: Namespace for the Job.
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
                        container=container,
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
                        pod, namespace, container=container
                    )
                    logging.info("The logs of pod %s:\n %s", pod, pod_logs)
                except Exception:
                    raise RuntimeError(
                        f"Failed to read logs for pod {namespace}/{pod.metadata.name}"
                    )

    # ------------------------------------------------------------------------ #
    # TFJob Training Client APIs.
    # ------------------------------------------------------------------------ #
    def create_tfjob(
        self,
        tfjob: models.KubeflowOrgV1TFJob,
        namespace=utils.get_default_target_namespace(),
    ):
        """Create the TFJob.

        Args:
            tfjob: TFJob object of type KubeflowOrgV1TFJob.
            namespace: Namespace for the TFJob.

        Raises:
            TimeoutError: Timeout to create TFJob.
            RuntimeError: Failed to create TFJob.
        """

        utils.create_job(
            custom_api=self.custom_api,
            job=tfjob,
            namespace=namespace,
            job_kind=constants.TFJOB_KIND,
            job_plural=constants.TFJOB_PLURAL,
        )

    def create_tfjob_from_func(
        self,
        name: str,
        func: Callable,
        parameters: Dict[str, Any] = None,
        base_image: str = constants.TFJOB_BASE_IMAGE,
        namespace: str = utils.get_default_target_namespace(),
        num_chief_replicas: int = None,
        num_ps_replicas: int = None,
        num_worker_replicas: int = None,
        packages_to_install: List[str] = None,
        pip_index_url: str = "https://pypi.org/simple",
    ):
        """Create TFJob from the function.

        Args:
            name: Name for the TFJob.
            func: Function that TFJob uses to train the model. This function
                must be Callable. Optionally, this function might have one dict
                argument to define input parameters for the function.
            parameters: Dict of input parameters that training function might receive.
            base_image: Image to use when executing the training function.
            namespace: Namespace for the TFJob.
            num_chief_replicas: Number of Chief replicas for the TFJob. Number
                of Chief replicas can't be more than 1.
            num_ps_replicas: Number of Parameter Server replicas for the TFJob.
            num_worker_replicas: Number of Worker replicas for the TFJob.
            packages_to_install: List of Python packages to install in addition
                to the base image packages. These packages are installed before
                executing the objective function.
            pip_index_url: The PyPI url from which to install Python packages.

        Raises:
            ValueError: TFJob replicas are missing or training function is invalid.
            TimeoutError: Timeout to create TFJob.
            RuntimeError: Failed to create TFJob.
        """

        # Check if at least one replica is set.
        # TODO (andreyvelich): Remove this check once we have CEL validation.
        # Ref: https://github.com/kubeflow/training-operator/issues/1708
        if (
            num_chief_replicas is None
            and num_ps_replicas is None
            and num_worker_replicas is None
        ):
            raise ValueError("At least one replica for TFJob must be set")

        # Check if function is callable.
        if not callable(func):
            raise ValueError(
                f"Training function must be callable, got function type: {type(func)}"
            )

        # Get TFJob Pod template spec.
        pod_template_spec = utils.get_pod_template_spec(
            func=func,
            parameters=parameters,
            base_image=base_image,
            container_name=constants.TFJOB_CONTAINER,
            packages_to_install=packages_to_install,
            pip_index_url=pip_index_url,
        )

        # Create TFJob template.
        tfjob = models.KubeflowOrgV1TFJob(
            api_version=f"{constants.KUBEFLOW_GROUP}/{constants.OPERATOR_VERSION}",
            kind=constants.TFJOB_KIND,
            metadata=client.V1ObjectMeta(name=name, namespace=namespace),
            spec=models.KubeflowOrgV1TFJobSpec(
                run_policy=models.V1RunPolicy(clean_pod_policy=None),
                tf_replica_specs={},
            ),
        )

        # Add Chief, PS, and Worker replicas to the TFJob.
        if num_chief_replicas is not None:
            tfjob.spec.tf_replica_specs[
                constants.REPLICA_TYPE_CHIEF
            ] = models.V1ReplicaSpec(
                replicas=num_chief_replicas, template=pod_template_spec,
            )

        if num_ps_replicas is not None:
            tfjob.spec.tf_replica_specs[
                constants.REPLICA_TYPE_PS
            ] = models.V1ReplicaSpec(
                replicas=num_ps_replicas, template=pod_template_spec,
            )

        if num_worker_replicas is not None:
            tfjob.spec.tf_replica_specs[
                constants.REPLICA_TYPE_WORKER
            ] = models.V1ReplicaSpec(
                replicas=num_worker_replicas, template=pod_template_spec,
            )

        # Create TFJob.
        self.create_tfjob(tfjob=tfjob, namespace=namespace)

    def get_tfjob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Get the TFJob.

        Args:
            name: Name for the TFJob.
            namespace: Namespace for the TFJob.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            KubeflowOrgV1TFJob: TFJob object.

        Raises:
            TimeoutError: Timeout to get TFJob.
            RuntimeError: Failed to get TFJob.
        """

        return utils.get_job(
            custom_api=self.custom_api,
            api_client=self.api_client,
            name=name,
            namespace=namespace,
            job_model=models.KubeflowOrgV1TFJob,
            job_kind=constants.TFJOB_KIND,
            job_plural=constants.TFJOB_PLURAL,
            timeout=timeout,
        )

    def list_tfjobs(
        self,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """List of all TFJobs in namespace.

        Args:
            namespace: Namespace to list the TFJobs.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            list[KubeflowOrgV1TFJob]: List of TFJobs objects. It returns
            empty list if TFJobs cannot be found.

        Raises:
            TimeoutError: Timeout to list TFJobs.
            RuntimeError: Failed to list TFJobs.
        """

        return utils.list_jobs(
            custom_api=self.custom_api,
            api_client=self.api_client,
            namespace=namespace,
            job_model=models.KubeflowOrgV1TFJob,
            job_kind=constants.TFJOB_KIND,
            job_plural=constants.TFJOB_PLURAL,
            timeout=timeout,
        )

    def delete_tfjob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        delete_options: client.V1DeleteOptions = None,
    ):
        """Delete the TFJob

        Args:
            name: Name for the TFJob.
            namespace: Namespace for the TFJob.
            delete_options: Optional, V1DeleteOptions to set while deleting
                the TFJob. For example, grace period seconds.

        Raises:
            TimeoutError: Timeout to delete TFJob.
            RuntimeError: Failed to delete TFJob.
        """

        utils.delete_job(
            custom_api=self.custom_api,
            name=name,
            namespace=namespace,
            job_kind=constants.TFJOB_KIND,
            job_plural=constants.TFJOB_PLURAL,
            delete_options=delete_options,
        )

    def patch_tfjob(
        self,
        tfjob: models.KubeflowOrgV1TFJob,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
    ):
        """Patch the TFJob.

        Args:
            tfjob: TFJob object of type KubeflowOrgV1TFJob to patch.
            name: Name for the TFJob.
            namespace: Namespace for the TFJob.

        Raises:
            TimeoutError: Timeout to patch TFJob.
            RuntimeError: Failed to patch TFJob.
        """

        return utils.patch_job(
            custom_api=self.custom_api,
            job=tfjob,
            name=name,
            namespace=namespace,
            job_kind=constants.TFJOB_KIND,
            job_plural=constants.TFJOB_PLURAL,
        )

    # ------------------------------------------------------------------------ #
    # PyTorchJob Training Client APIs.
    # ------------------------------------------------------------------------ #
    def create_pytorchjob(
        self,
        pytorchjob: models.KubeflowOrgV1PyTorchJob,
        namespace=utils.get_default_target_namespace(),
    ):
        """Create the PyTorchJob.

        Args:
            pytorchjob: PyTorchJob object of type KubeflowOrgV1PyTorchJob.
            namespace: Namespace for the PyTorchJob.

        Raises:
            TimeoutError: Timeout to create PyTorchJob.
            RuntimeError: Failed to create PyTorchJob.
        """

        utils.create_job(
            custom_api=self.custom_api,
            job=pytorchjob,
            namespace=namespace,
            job_kind=constants.PYTORCHJOB_KIND,
            job_plural=constants.PYTORCHJOB_PLURAL,
        )

    def create_pytorchjob_from_func(
        self,
        name: str,
        func: Callable,
        parameters: Dict[str, Any] = None,
        base_image: str = constants.PYTORCHJOB_BASE_IMAGE,
        namespace: str = utils.get_default_target_namespace(),
        num_worker_replicas: int = None,
        packages_to_install: List[str] = None,
        pip_index_url: str = "https://pypi.org/simple",
    ):
        """Create PyTorchJob from the function.

        Args:
            name: Name for the PyTorchJob.
            func: Function that PyTorchJob uses to train the model. This function
                must be Callable. Optionally, this function might have one dict
                argument to define input parameters for the function.
            parameters: Dict of input parameters that training function might receive.
            base_image: Image to use when executing the training function.
            namespace: Namespace for the PyTorchJob.
            num_worker_replicas: Number of Worker replicas for the PyTorchJob.
                If number of Worker replicas is 1, PyTorchJob uses only
                Master replica.
            packages_to_install: List of Python packages to install in addition
                to the base image packages. These packages are installed before
                executing the objective function.
            pip_index_url: The PyPI url from which to install Python packages.
        """

        # Check if at least one worker replica is set.
        # TODO (andreyvelich): Remove this check once we have CEL validation.
        # Ref: https://github.com/kubeflow/training-operator/issues/1708
        if num_worker_replicas is None:
            raise ValueError("At least one Worker replica for PyTorchJob must be set")

        # Check if function is callable.
        if not callable(func):
            raise ValueError(
                f"Training function must be callable, got function type: {type(func)}"
            )

        # Get PyTorchJob Pod template spec.
        pod_template_spec = utils.get_pod_template_spec(
            func=func,
            parameters=parameters,
            base_image=base_image,
            container_name=constants.PYTORCHJOB_CONTAINER,
            packages_to_install=packages_to_install,
            pip_index_url=pip_index_url,
        )

        # Create PyTorchJob template.
        pytorchjob = models.KubeflowOrgV1PyTorchJob(
            api_version=f"{constants.KUBEFLOW_GROUP}/{constants.OPERATOR_VERSION}",
            kind=constants.PYTORCHJOB_KIND,
            metadata=client.V1ObjectMeta(name=name, namespace=namespace),
            spec=models.KubeflowOrgV1PyTorchJobSpec(
                run_policy=models.V1RunPolicy(clean_pod_policy=None),
                pytorch_replica_specs={},
            ),
        )

        # Add Master and Worker replicas to the PyTorchJob.
        pytorchjob.spec.pytorch_replica_specs[
            constants.REPLICA_TYPE_MASTER
        ] = models.V1ReplicaSpec(replicas=1, template=pod_template_spec,)

        # If number of Worker replicas is 1, PyTorchJob uses only Master replica.
        if num_worker_replicas != 1:
            pytorchjob.spec.pytorch_replica_specs[
                constants.REPLICA_TYPE_WORKER
            ] = models.V1ReplicaSpec(
                replicas=num_worker_replicas, template=pod_template_spec,
            )

        # Create PyTorchJob
        self.create_pytorchjob(pytorchjob=pytorchjob, namespace=namespace)

    def get_pytorchjob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Get the PyTorchJob.

        Args:
            name: Name for the PyTorchJob.
            namespace: Namespace for the PyTorchJob.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            KubeflowOrgV1PyTorchJob: PyTorchJob object.

        Raises:
            TimeoutError: Timeout to get PyTorchJob.
            RuntimeError: Failed to get PyTorchJob.
        """

        return utils.get_job(
            custom_api=self.custom_api,
            api_client=self.api_client,
            name=name,
            namespace=namespace,
            job_model=models.KubeflowOrgV1PyTorchJob,
            job_kind=constants.PYTORCHJOB_KIND,
            job_plural=constants.PYTORCHJOB_PLURAL,
            timeout=timeout,
        )

    def list_pytorchjobs(
        self,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """List of all PyTorchJob in namespace.

        Args:
            namespace: Namespace to list the PyTorchJob.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            list[KubeflowOrgV1PyTorchJob]: List of PyTorchJob objects. It returns
            empty list if PyTorchJobs cannot be found.

        Raises:
            TimeoutError: Timeout to list PyTorchJobs.
            RuntimeError: Failed to list PyTorchJobs.
        """

        return utils.list_jobs(
            custom_api=self.custom_api,
            api_client=self.api_client,
            namespace=namespace,
            job_model=models.KubeflowOrgV1PyTorchJob,
            job_kind=constants.PYTORCHJOB_KIND,
            job_plural=constants.PYTORCHJOB_PLURAL,
            timeout=timeout,
        )

    def delete_pytorchjob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        delete_options: client.V1DeleteOptions = None,
    ):
        """Delete the PyTorchJob

        Args:
            name: Name for the PyTorchJob.
            namespace: Namespace for the PyTorchJob.
            delete_options: Optional, V1DeleteOptions to set while deleting
                the PyTorchJob. For example, grace period seconds.

        Raises:
            TimeoutError: Timeout to delete PyTorchJob.
            RuntimeError: Failed to delete PyTorchJob.
        """

        utils.delete_job(
            custom_api=self.custom_api,
            name=name,
            namespace=namespace,
            job_kind=constants.PYTORCHJOB_KIND,
            job_plural=constants.PYTORCHJOB_PLURAL,
            delete_options=delete_options,
        )

    def patch_pytorchjob(
        self,
        pytorchjob: models.KubeflowOrgV1PyTorchJob,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
    ):
        """Patch the PyTorchJob.

        Args:
            pytorchjob: PyTorchJob object of type KubeflowOrgV1PyTorchJob.
            name: Name for the PyTorchJob.
            namespace: Namespace for the PyTorchJob.

        Raises:
            TimeoutError: Timeout to patch PyTorchJob.
            RuntimeError: Failed to patch PyTorchJob.
        """

        return utils.patch_job(
            custom_api=self.custom_api,
            job=pytorchjob,
            name=name,
            namespace=namespace,
            job_kind=constants.PYTORCHJOB_KIND,
            job_plural=constants.PYTORCHJOB_PLURAL,
        )

    # ------------------------------------------------------------------------ #
    # MXJob Training Client APIs.
    # ------------------------------------------------------------------------ #
    def create_mxjob(
        self,
        mxjob: models.KubeflowOrgV1MXJob,
        namespace=utils.get_default_target_namespace(),
    ):
        """Create the MXJob.

        Args:
            mxjob: MXJob object of type KubeflowOrgV1MXJob.
            namespace: Namespace for the MXJob.

        Raises:
            TimeoutError: Timeout to create MXJob.
            RuntimeError: Failed to create MXJob.
        """

        utils.create_job(
            custom_api=self.custom_api,
            job=mxjob,
            namespace=namespace,
            job_kind=constants.MXJOB_KIND,
            job_plural=constants.MXJOB_PLURAL,
        )

    def create_mxjob_from_func(self):
        """Create MXJob from the function.
        TODO (andreyvelich): Implement this function.
        """
        logging.warning("This API has not been implemented yet.")

    def get_mxjob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Get the MXJob.

        Args:
            name: Name for the MXJob.
            namespace: Namespace for the MXJob.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            KubeflowOrgV1MXJob: MXJob object.

        Raises:
            TimeoutError: Timeout to get MXJob.
            RuntimeError: Failed to get MXJob.
        """

        return utils.get_job(
            custom_api=self.custom_api,
            api_client=self.api_client,
            name=name,
            namespace=namespace,
            job_model=models.KubeflowOrgV1MXJob,
            job_kind=constants.MXJOB_KIND,
            job_plural=constants.MXJOB_PLURAL,
            timeout=timeout,
        )

    def list_mxjobs(
        self,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """List of all MXJobs in namespace.

        Args:
            namespace: Namespace to list the MXJobs.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            list[KubeflowOrgV1MXJob]: List of MXJobs objects. It returns
            empty list if MXJobs cannot be found.

        Raises:
            TimeoutError: Timeout to list MXJobs.
            RuntimeError: Failed to list MXJobs.
        """

        return utils.list_jobs(
            custom_api=self.custom_api,
            api_client=self.api_client,
            namespace=namespace,
            job_model=models.KubeflowOrgV1MXJob,
            job_kind=constants.MXJOB_KIND,
            job_plural=constants.MXJOB_PLURAL,
            timeout=timeout,
        )

    def delete_mxjob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        delete_options: client.V1DeleteOptions = None,
    ):
        """Delete the MXJob

        Args:
            name: Name for the MXJob.
            namespace: Namespace for the MXJob.
            delete_options: Optional, V1DeleteOptions to set while deleting
                the MXJob. For example, grace period seconds.

        Raises:
            TimeoutError: Timeout to delete MXJob.
            RuntimeError: Failed to delete MXJob.
        """

        utils.delete_job(
            custom_api=self.custom_api,
            name=name,
            namespace=namespace,
            job_kind=constants.MXJOB_KIND,
            job_plural=constants.MXJOB_PLURAL,
            delete_options=delete_options,
        )

    def patch_mxjob(
        self,
        mxjob: models.KubeflowOrgV1MXJob,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
    ):
        """Patch the MXJob.

        Args:
            mxjob: MXJob object of type KubeflowOrgV1MXJob.
            name: Name for the MXJob.
            namespace: Namespace for the MXJob.

        Raises:
            TimeoutError: Timeout to patch MXJob.
            RuntimeError: Failed to patch MXJob.
        """

        return utils.patch_job(
            custom_api=self.custom_api,
            job=mxjob,
            name=name,
            namespace=namespace,
            job_kind=constants.MXJOB_KIND,
            job_plural=constants.MXJOB_PLURAL,
        )

    # ------------------------------------------------------------------------ #
    # XGBoostJob Training Client APIs.
    # ------------------------------------------------------------------------ #
    def create_xgboostjob(
        self,
        xgboostjob: models.KubeflowOrgV1XGBoostJob,
        namespace=utils.get_default_target_namespace(),
    ):
        """Create the XGBoostJob.

        Args:
            xgboostjob: XGBoostJob object of type KubeflowOrgV1XGBoostJob.
            namespace: Namespace for the XGBoostJob.

        Raises:
            TimeoutError: Timeout to create XGBoostJob.
            RuntimeError: Failed to create XGBoostJob.
        """

        utils.create_job(
            custom_api=self.custom_api,
            job=xgboostjob,
            namespace=namespace,
            job_kind=constants.XGBOOSTJOB_KIND,
            job_plural=constants.XGBOOSTJOB_PLURAL,
        )

    def create_xgboostjob_from_func(self):
        """Create XGBoost from the function.
        TODO (andreyvelich): Implement this function.
        """
        logging.warning("This API has not been implemented yet.")

    def get_xgboostjob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Get the XGBoostJob.

        Args:
            name: Name for the XGBoostJob.
            namespace: Namespace for the XGBoostJob.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            KubeflowOrgV1XGBoostJob: XGBoostJob object.

        Raises:
            TimeoutError: Timeout to get XGBoostJob.
            RuntimeError: Failed to get XGBoostJob.
        """

        return utils.get_job(
            custom_api=self.custom_api,
            api_client=self.api_client,
            name=name,
            namespace=namespace,
            job_model=models.KubeflowOrgV1XGBoostJob,
            job_kind=constants.XGBOOSTJOB_KIND,
            job_plural=constants.XGBOOSTJOB_PLURAL,
            timeout=timeout,
        )

    def list_xgboostjobs(
        self,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """List of all XGBoostJobs in namespace.

        Args:
            namespace: Namespace to list the XGBoostJobs.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            list[KubeflowOrgV1XGBoostJob]: List of XGBoostJobs objects. It returns
            empty list if XGBoostJobs cannot be found.

        Raises:
            TimeoutError: Timeout to list XGBoostJobs.
            RuntimeError: Failed to list XGBoostJobs.
        """

        return utils.list_jobs(
            custom_api=self.custom_api,
            api_client=self.api_client,
            namespace=namespace,
            job_model=models.KubeflowOrgV1XGBoostJob,
            job_kind=constants.XGBOOSTJOB_KIND,
            job_plural=constants.XGBOOSTJOB_PLURAL,
            timeout=timeout,
        )

    def delete_xgboostjob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        delete_options: client.V1DeleteOptions = None,
    ):
        """Delete the XGBoostJob

        Args:
            name: Name for the XGBoostJob.
            namespace: Namespace for the XGBoostJob.
            delete_options: Optional, V1DeleteOptions to set while deleting
                the XGBoostJob. For example, grace period seconds.

        Raises:
            TimeoutError: Timeout to delete XGBoostJob.
            RuntimeError: Failed to delete XGBoostJob.
        """

        utils.delete_job(
            custom_api=self.custom_api,
            name=name,
            namespace=namespace,
            job_kind=constants.XGBOOSTJOB_KIND,
            job_plural=constants.XGBOOSTJOB_PLURAL,
            delete_options=delete_options,
        )

    def patch_xgboostjob(
        self,
        xgboostjob: models.KubeflowOrgV1XGBoostJob,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
    ):
        """Patch the XGBoostJob.

        Args:
            xgboostjob: XGBoostJob object of type KubeflowOrgV1XGBoostJob.
            name: Name for the XGBoostJob.
            namespace: Namespace for the XGBoostJob.

        Raises:
            TimeoutError: Timeout to patch XGBoostJob.
            RuntimeError: Failed to patch XGBoostJob.
        """

        return utils.patch_job(
            custom_api=self.custom_api,
            job=xgboostjob,
            name=name,
            namespace=namespace,
            job_kind=constants.XGBOOSTJOB_KIND,
            job_plural=constants.XGBOOSTJOB_PLURAL,
        )

    # ------------------------------------------------------------------------ #
    # MPIJob Training Client APIs.
    # ------------------------------------------------------------------------ #
    def create_mpijob(
        self,
        mpijob: models.KubeflowOrgV1MPIJob,
        namespace=utils.get_default_target_namespace(),
    ):
        """Create the MPIJob.

        Args:
            mpijob: MPIJob object of type KubeflowOrgV1MPIJob.
            namespace: Namespace for the MPIJob.

        Raises:
            TimeoutError: Timeout to create MPIJob.
            RuntimeError: Failed to create MPIJob.
        """

        utils.create_job(
            custom_api=self.custom_api,
            job=mpijob,
            namespace=namespace,
            job_kind=constants.MPIJOB_KIND,
            job_plural=constants.MPIJOB_PLURAL,
        )

    def create_mpijob_from_func(self):
        """Create MPIJob from the function.
        TODO (andreyvelich): Implement this function.
        """
        logging.warning("This API has not been implemented yet.")

    def get_mpijob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Get the MPIJob.

        Args:
            name: Name for the MPIJob.
            namespace: Namespace for the MPIJob.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            KubeflowOrgV1MPIJob: MPIJob object.

        Raises:
            TimeoutError: Timeout to get MPIJob.
            RuntimeError: Failed to get MPIJob.
        """

        return utils.get_job(
            custom_api=self.custom_api,
            api_client=self.api_client,
            name=name,
            namespace=namespace,
            job_model=models.KubeflowOrgV1MPIJob,
            job_kind=constants.MPIJOB_KIND,
            job_plural=constants.MPIJOB_PLURAL,
            timeout=timeout,
        )

    def list_mpijobs(
        self,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """List of all MPIJobs in namespace.

        Args:
            namespace: Namespace to list the MPIJobs.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            list[KubeflowOrgV1MPIJob]: List of MPIJobs objects. It returns
            empty list if MPIJobs cannot be found.

        Raises:
            TimeoutError: Timeout to list MPIJobs.
            RuntimeError: Failed to list MPIJobs.
        """

        return utils.list_jobs(
            custom_api=self.custom_api,
            api_client=self.api_client,
            namespace=namespace,
            job_model=models.KubeflowOrgV1MPIJob,
            job_kind=constants.MPIJOB_KIND,
            job_plural=constants.MPIJOB_PLURAL,
            timeout=timeout,
        )

    def delete_mpijob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        delete_options: client.V1DeleteOptions = None,
    ):
        """Delete the MPIJob

        Args:
            name: Name for the MPIJob.
            namespace: Namespace for the MPIJob.
            delete_options: Optional, V1DeleteOptions to set while deleting
                the MPIJob. For example, grace period seconds.

        Raises:
            TimeoutError: Timeout to delete MPIJob.
            RuntimeError: Failed to delete MPIJob.
        """

        utils.delete_job(
            custom_api=self.custom_api,
            name=name,
            namespace=namespace,
            job_kind=constants.MPIJOB_KIND,
            job_plural=constants.MPIJOB_PLURAL,
            delete_options=delete_options,
        )

    def patch_mpijob(
        self,
        mpijob: models.KubeflowOrgV1MPIJob,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
    ):
        """Patch the MPIJob.

        Args:
            mpijob: MPIJob object of type KubeflowOrgV1MPIJob.
            name: Name for the MPIJob.
            namespace: Namespace for the MPIJob.

        Raises:
            TimeoutError: Timeout to patch MPIJob.
            RuntimeError: Failed to patch MPIJob.
        """

        return utils.patch_job(
            custom_api=self.custom_api,
            job=mpijob,
            name=name,
            namespace=namespace,
            job_kind=constants.MPIJOB_KIND,
            job_plural=constants.MPIJOB_PLURAL,
        )

    # ------------------------------------------------------------------------ #
    # PaddleJob Training Client APIs.
    # ------------------------------------------------------------------------ #
    def create_paddlejob(
        self,
        paddlejob: models.KubeflowOrgV1PaddleJob,
        namespace=utils.get_default_target_namespace(),
    ):
        """Create the PaddleJob.

        Args:
            paddlejob: PaddleJob object of type KubeflowOrgV1PaddleJob.
            namespace: Namespace for the PaddleJob.

        Raises:
            TimeoutError: Timeout to create PaddleJob.
            RuntimeError: Failed to create PaddleJob.
        """

        utils.create_job(
            custom_api=self.custom_api,
            job=paddlejob,
            namespace=namespace,
            job_kind=constants.PADDLEJOB_KIND,
            job_plural=constants.PADDLEJOB_PLURAL,
        )

    def create_paddlejob_from_func(self):
        """Create PaddleJob from the function.
        TODO (andreyvelich): Implement this function.
        """
        logging.warning("This API has not been implemented yet.")

    def get_paddlejob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """Get the PaddleJob.

        Args:
            name: Name for the PaddleJob.
            namespace: Namespace for the PaddleJob.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            KubeflowOrgV1PaddleJob: PaddleJob object.

        Raises:
            TimeoutError: Timeout to get PaddleJob.
            RuntimeError: Failed to get PaddleJob.
        """

        return utils.get_job(
            custom_api=self.custom_api,
            api_client=self.api_client,
            name=name,
            namespace=namespace,
            job_model=models.KubeflowOrgV1PaddleJob,
            job_kind=constants.PADDLEJOB_KIND,
            job_plural=constants.PADDLEJOB_PLURAL,
            timeout=timeout,
        )

    def list_paddlejobs(
        self,
        namespace: str = utils.get_default_target_namespace(),
        timeout: int = constants.DEFAULT_TIMEOUT,
    ):
        """List of all PaddleJobs in namespace.

        Args:
            namespace: Namespace to list the PaddleJobs.
            timeout: Optional, Kubernetes API server timeout in seconds
                to execute the request.

        Returns:
            list[KubeflowOrgV1PaddleJob]: List of PaddleJobs objects. It returns
            empty list if PaddleJobs cannot be found.

        Raises:
            TimeoutError: Timeout to list PaddleJobs.
            RuntimeError: Failed to list PaddleJobs.
        """

        return utils.list_jobs(
            custom_api=self.custom_api,
            api_client=self.api_client,
            namespace=namespace,
            job_model=models.KubeflowOrgV1PaddleJob,
            job_kind=constants.PADDLEJOB_KIND,
            job_plural=constants.PADDLEJOB_PLURAL,
            timeout=timeout,
        )

    def delete_paddlejob(
        self,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
        delete_options: client.V1DeleteOptions = None,
    ):
        """Delete the PaddleJob

        Args:
            name: Name for the PaddleJob.
            namespace: Namespace for the PaddleJob.
            delete_options: Optional, V1DeleteOptions to set while deleting
                the PaddleJob. For example, grace period seconds.

        Raises:
            TimeoutError: Timeout to delete PaddleJob.
            RuntimeError: Failed to delete PaddleJob.
        """

        utils.delete_job(
            custom_api=self.custom_api,
            name=name,
            namespace=namespace,
            job_kind=constants.PADDLEJOB_KIND,
            job_plural=constants.PADDLEJOB_PLURAL,
            delete_options=delete_options,
        )

    def patch_paddlejob(
        self,
        paddlejob: models.KubeflowOrgV1PaddleJob,
        name: str,
        namespace: str = utils.get_default_target_namespace(),
    ):
        """Patch the PaddleJob.

        Args:
            paddlejob: PaddleJob object of type KubeflowOrgV1PaddleJob.
            name: Name for the PaddleJob.
            namespace: Namespace for the PaddleJob.

        Raises:
            TimeoutError: Timeout to patch PaddleJob.
            RuntimeError: Failed to patch PaddleJob.
        """

        return utils.patch_job(
            custom_api=self.custom_api,
            job=paddlejob,
            name=name,
            namespace=namespace,
            job_kind=constants.PADDLEJOB_KIND,
            job_plural=constants.PADDLEJOB_PLURAL,
        )
