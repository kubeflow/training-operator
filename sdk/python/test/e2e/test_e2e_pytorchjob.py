# Copyright 2021 kubeflow.org.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os

from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container

from kubeflow.training import PyTorchJobClient
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1PyTorchJob
from kubeflow.training import KubeflowOrgV1PyTorchJobSpec
from kubeflow.training import V1RunPolicy

PYTORCH_CLIENT = PyTorchJobClient(config_file=os.getenv('KUBECONFIG', '~/.kube/config'))
SDK_TEST_NAMESPACE = 'default'


def test_sdk_e2e():
    container = V1Container(
        name="pytorch",
        image="gcr.io/kubeflow-ci/pytorch-dist-mnist-test:v1.0",
        args=["--backend", "gloo"],
    )

    master = V1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(
            spec=V1PodSpec(
                containers=[container]
            )
        )
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(
            spec=V1PodSpec(
                containers=[container]
            )
        )
    )

    pytorchjob = KubeflowOrgV1PyTorchJob(
        api_version="kubeflow.org/v1",
        kind="PyTorchJob",
        metadata=V1ObjectMeta(name="pytorchjob-mnist-ci-test", namespace=SDK_TEST_NAMESPACE),
        spec=KubeflowOrgV1PyTorchJobSpec(
            run_policy=V1RunPolicy(
                clean_pod_policy="None",
            ),
            pytorch_replica_specs={"Master": master,
                                   "Worker": worker}
        )
    )

    PYTORCH_CLIENT.create(pytorchjob)

    PYTORCH_CLIENT.wait_for_job("pytorchjob-mnist-ci-test", namespace=SDK_TEST_NAMESPACE)
    if not PYTORCH_CLIENT.is_job_succeeded("pytorchjob-mnist-ci-test",
                                           namespace=SDK_TEST_NAMESPACE):
        raise RuntimeError("The PyTorchJob is not succeeded.")

    PYTORCH_CLIENT.get_logs("pytorchjob-mnist-ci-test", namespace=SDK_TEST_NAMESPACE)

    PYTORCH_CLIENT.delete("pytorchjob-mnist-ci-test", namespace=SDK_TEST_NAMESPACE)
