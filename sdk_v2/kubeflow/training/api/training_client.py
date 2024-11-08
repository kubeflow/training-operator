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
from typing import List, Optional

from kubeflow.training import models
from kubeflow.training.api_client import ApiClient
from kubeflow.training.constants import constants
from kubeflow.training.utils import utils
from kubernetes import client, config

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
        # self.core_api = client.CoreV1Api(k8s_client)
        self.api_client = ApiClient()

        self.namespace = namespace

    def list_jobs(
        self,
        timeout: int = constants.DEFAULT_TIMEOUT,
    ) -> List[models.KubeflowOrgV2alpha1TrainJob]:
        """List of all TrainJobs.

        Args:
            timeout: Kubernetes API server timeout in seconds to execute the request.

        Returns:
            List[KubeflowOrgV2alpha1TrainJob]: List of TrainJob objects.
                For example: list of KubeflowOrgV2alpha1TrainJob objects. It returns empty list
                if TrainJobs can't be found.

        Raises:
            TimeoutError: Timeout to list TrainJobs
            RuntimeError: Failed to list TrainJobs
        """

        result = []
        try:
            thread = self.custom_api.list_namespaced_custom_object(
                constants.GROUP,
                constants.VERSION,
                self.namespace,
                constants.TRAIN_JOB_PLURAL,
            )
            response = thread.get(timeout)
            if response is not None:
                result = [
                    self.api_client.deserialize(
                        utils.FakeResponse(item),
                        models.KubeflowOrgV2alpha1TrainJob.__name__,
                    )
                    for item in response.get("items")
                ]
        except multiprocessing.TimeoutError:
            raise TimeoutError(
                f"Timeout to list {constants.TRAIN_JOB_KIND}s in namespace: {self.namespace}"
            )
        except Exception:
            raise RuntimeError(
                f"Failed to list {constants.TRAIN_JOB_KIND}s in namespace: {self.namespace}"
            )

        return result  # type: ignore

    def get_runtimes(self):
        pass

    def train(
        self,
        train_func,
        parameters,
        num_nodes,
        scaling_config,
        resources_per_nodes,
        runtime_ref: str,
    ):
        pass

    def get_torch_runtimes(self):
        pass

    def get_llm_runtimes(self):
        pass
