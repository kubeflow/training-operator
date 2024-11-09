# Copyright 2024 The Kubeflow Authors.
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

import logging
import multiprocessing
import queue
import random
import string
import uuid
from typing import Callable, Dict, List, Optional

from kubeflow.training import models
from kubeflow.training.api_client import ApiClient
from kubeflow.training.constants import constants
from kubeflow.training.types import types
from kubeflow.training.utils import utils
from kubernetes import client, config, watch

logger = logging.getLogger(__name__)


class TrainingClient:
    def __init__(
        self,
        config_file: Optional[str] = None,
        context: Optional[str] = None,
        client_configuration: Optional[client.Configuration] = None,
        namespace: str = utils.get_default_target_namespace(),
    ):
        """TrainingClient constructor. Configure logging in your application
            as follows to see detailed information from the TrainingClient APIs:
            .. code-block:: python
                import logging
                logging.basicConfig()
                log = logging.getLogger("kubeflow.training.api.training_client")
                log.setLevel(logging.DEBUG)

        Args:
            config_file: Path to the kube-config file. Defaults to ~/.kube/config.
            context: Set the active context. Defaults to current_context from the kube-config.
            client_configuration: Client configuration for cluster authentication.
                You have to provide valid configuration with Bearer token or
                with username and password. You can find an example here:
                https://github.com/kubernetes-client/python/blob/67f9c7a97081b4526470cad53576bc3b71fa6fcc/examples/remote_cluster.py#L31
            namespace: Target Kubernetes namespace. If SDK runs outside of Kubernetes cluster it
                takes the namespace from the kube-config context. If SDK runs inside
                the Kubernetes cluster it takes namespace from the
                `/var/run/secrets/kubernetes.io/serviceaccount/namespace` file. By default it
                uses the `default` namespace.
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

    # TODO (andreyvelich): Currently, only Cluster Training Runtime is supported.
    def list_runtimes(self) -> List[types.Runtime]:
        """List of the available runtimes.

        Returns:
            List[Runtime]: List of available training runtimes. It returns an empty list if
                runtimes don't exist.

        Raises:
            TimeoutError: Timeout to list Runtimes.
            RuntimeError: Failed to list Runtimes.
        """

        result = []
        try:
            thread = self.custom_api.list_cluster_custom_object(
                constants.GROUP,
                constants.VERSION,
                constants.CLUSTER_TRAINING_RUNTIME_PLURAL,
                async_req=True,
            )
            response = thread.get(constants.DEFAULT_TIMEOUT)
            for item in response["items"]:
                # TODO (andreyvelich): Currently, the training phase label must be presented.
                if "labels" in item["metadata"]:
                    result.append(
                        types.Runtime(
                            name=item["metadata"]["name"],
                            phase=item["metadata"]["labels"][constants.PHASE_KEY],
                        )
                    )
        except multiprocessing.TimeoutError:
            raise TimeoutError(
                f"Timeout to list {constants.CLUSTER_TRAINING_RUNTIME_KIND}s "
                f"in namespace: {self.namespace}"
            )
        except Exception as e:
            raise RuntimeError(
                f"Failed to list {constants.CLUSTER_TRAINING_RUNTIME_KIND}s "
                f"in namespace: {self.namespace}. Error: {e}"
            )

        return result

    def train(
        self,
        runtime_ref: str,
        train_func: Optional[Callable] = None,
        num_nodes: Optional[int] = None,
        resources_per_node: Optional[dict] = None,
    ) -> str:
        """Create the TrainJob. TODO: Add description

        Returns:
            str: The unique name of the TrainJob that has been generated.

        Raises:
            ValueError: Input arguments are invalid.
            TimeoutError: Timeout to create TrainJobs.
            RuntimeError: Failed to create TrainJobs.
        """

        # Generate unique name for the TrainJob.
        # TODO (andreyvelich): Discuss this TrainJob name generation.
        train_job_name = random.choice(string.ascii_lowercase) + uuid.uuid4().hex[:11]

        trainer = models.KubeflowOrgV2alpha1Trainer()
        # Add number of nodes to the Trainer.
        if num_nodes is not None:
            trainer.num_nodes = num_nodes

        # Add resources per node to the Trainer.
        if resources_per_node is not None:
            trainer.resources_per_node = utils.get_resources_per_node(
                resources_per_node
            )

        if train_func is not None:
            trainer.command = constants.DEFAULT_COMMAND
            # TODO: Support more params
            trainer.args = utils.get_args_using_train_func(train_func)

        train_job = models.KubeflowOrgV2alpha1TrainJob(
            api_version=constants.API_VERSION,
            kind=constants.TRAINJOB_KIND,
            metadata=client.V1ObjectMeta(name=train_job_name),
            spec=models.KubeflowOrgV2alpha1TrainJobSpec(
                runtime_ref=models.KubeflowOrgV2alpha1RuntimeRef(name=runtime_ref),
                trainer=(
                    trainer if trainer != models.KubeflowOrgV2alpha1Trainer() else None
                ),
            ),
        )

        # Create the TrainJob.
        try:
            self.custom_api.create_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                self.namespace,
                constants.TRAINJOB_PLURAL,
                train_job,
            )
        except multiprocessing.TimeoutError:
            raise TimeoutError(
                f"Timeout to create {constants.TRAINJOB_KIND}: {self.namespace}/{train_job_name}"
            )
        except Exception:
            raise RuntimeError(
                f"Failed to create {constants.TRAINJOB_KIND}: {self.namespace}/{train_job_name}"
            )

        logger.debug(
            f"{constants.TRAINJOB_KIND} {self.namespace}/{train_job_name} has been created"
        )

        return train_job_name

    def list_jobs(self) -> List[types.TrainJob]:
        """List of all TrainJobs.

        Returns:
            List[KubeflowOrgV2alpha1TrainJob]: List of created TrainJobs. It returns an empty list
                if TrainJobs don't exist.

        Raises:
            TimeoutError: Timeout to list TrainJobs.
            RuntimeError: Failed to list TrainJobs.
        """

        result = []
        try:
            thread = self.custom_api.list_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                self.namespace,
                constants.TRAINJOB_PLURAL,
                async_req=True,
            )
            response = thread.get(constants.DEFAULT_TIMEOUT)

            for item in response["items"]:
                result.append(
                    types.TrainJob(
                        name=item["metadata"]["name"],
                        runtime_ref=item["spec"]["runtimeRef"]["name"],
                        creation_timestamp=item["metadata"]["creationTimestamp"],
                    )
                )
                # TODO: Add status
                if "status" in item:
                    pass

        except multiprocessing.TimeoutError:
            raise TimeoutError(
                f"Timeout to list {constants.TRAINJOB_KIND}s in namespace: {self.namespace}"
            )
        except Exception:
            raise RuntimeError(
                f"Failed to list {constants.TRAINJOB_KIND}s in namespace: {self.namespace}"
            )

        return result

    # TODO (andreyvelich): Discuss whether we need this API.
    # Potentially, we can move this data to the TrainJob type.
    def get_job_pods(self, name: str) -> List[types.Pod]:
        """Get pod names for the TrainJob Job."""

        result = []
        try:
            thread = self.core_api.list_namespaced_pod(
                self.namespace,
                label_selector=f"{constants.JOBSET_NAME_KEY}={name}",
                async_req=True,
            )
            response = thread.get(constants.DEFAULT_TIMEOUT)

            for item in response.items:
                result.append(
                    types.Pod(
                        name=item.metadata.name,
                        type=utils.get_pod_type(item.metadata.labels),
                        status=item.status.phase if item.status else None,
                    )
                )

        except multiprocessing.TimeoutError:
            raise TimeoutError(
                f"Timeout to list {constants.TRAINJOB_KIND}'s pods: {self.namespace}/{name}"
            )
        except Exception:
            raise RuntimeError(
                f"Failed to list {constants.TRAINJOB_KIND}'s pods: {self.namespace}/{name}"
            )

        return result

    def get_job_logs(
        self, name: str, follow: bool = False, stage: str = "trainer"
    ) -> Dict[str, str]:
        """Get the Trainer logs from TrainJob"""

        # Trainer node with index 0 must be deployed.
        pod = None
        for p in self.get_job_pods(name):
            if p.type == constants.MASTER_NODE and p.type != constants.POD_PENDING:
                pod = p
        if pod is None:
            return {}

        # Dict where key is the Pod type and value is the Pod logs.
        logs_dict = {}

        # TODO (andreyvelich): Potentially, refactor this.
        # Support logging of multiple Pods.
        if follow:
            log_streams = []
            log_streams.append(
                watch.Watch().stream(
                    self.core_api.read_namespaced_pod_log,
                    name=pod.name,
                    namespace=self.namespace,
                    container=constants.CONTAINER_TRAINER,
                )
            )
            finished = [False for _ in log_streams]

            # Create thread and queue per stream, for non-blocking iteration.
            log_queue_pool = utils.get_log_queue_pool(log_streams)

            # Iterate over every watching pods' log queue
            while True:
                for index, log_queue in enumerate(log_queue_pool):
                    if all(finished):
                        break
                    if finished[index]:
                        continue
                    # grouping the every 50 log lines of the same pod.
                    for _ in range(50):
                        try:
                            logline = log_queue.get(timeout=1)
                            if logline is None:
                                finished[index] = True
                                break
                            # Print logs to the StdOut
                            print(f"[{pod.type}]: {logline}")
                            # Add logs to the results dict.
                            if pod.type not in logs_dict:
                                logs_dict[pod.type] = logline + "\n"
                            else:
                                logs_dict[pod.type] += logline + "\n"
                        except queue.Empty:
                            break
                if all(finished):
                    return logs_dict

        try:
            pod_logs = self.core_api.read_namespaced_pod_log(
                name=pod.name,
                namespace=self.namespace,
                container=constants.CONTAINER_TRAINER,
            )
            logs_dict[pod.name] = pod_logs
        except Exception:
            raise RuntimeError(
                f"Failed to read logs for the pod {self.namespace}/{pod.name}"
            )

        return logs_dict

    def delete_job(self, name: str):
        """Delete the TrainJob.

        Args:
            name: Name of the TrainJob.

        Raises:
            TimeoutError: Timeout to delete TrainJob.
            RuntimeError: Failed to delete TrainJob.
        """

        try:
            self.custom_api.delete_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                self.namespace,
                constants.TRAINJOB_PLURAL,
                name=name,
            )
        except multiprocessing.TimeoutError:
            raise TimeoutError(
                f"Timeout to delete {constants.TRAINJOB_KIND}: {self.namespace}/{name}"
            )
        except Exception:
            raise RuntimeError(
                f"Failed to delete {constants.TRAINJOB_KIND}: {self.namespace}/{name}"
            )

        logger.debug(
            f"{constants.TRAINJOB_KIND} {self.namespace}/{name} has been deleted"
        )
