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

from kubeflow.training import TrainingClient
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1PyTorchJob
from kubeflow.training import KubeflowOrgV1PyTorchJobSpec
from kubeflow.training import V1RunPolicy
from kubeflow.training.constants import constants

from test.e2e.utils import verify_job_e2e

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "pytorchjob-mnist-ci-test"
JOB_NAMESPACE = "default"
CONTAINER_NAME = "pytorch"


def test_sdk_e2e():
    container = V1Container(
        name=CONTAINER_NAME,
        image="gcr.io/kubeflow-ci/pytorch-dist-mnist-test:v1.0",
        args=["--backend", "gloo"],
    )

    master = V1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[container])),
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[container])),
    )

    pytorchjob = KubeflowOrgV1PyTorchJob(
        api_version="kubeflow.org/v1",
        kind="PyTorchJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1PyTorchJobSpec(
            run_policy=V1RunPolicy(clean_pod_policy="None",),
            pytorch_replica_specs={"Master": master, "Worker": worker},
        ),
    )

    TRAINING_CLIENT.create_pytorchjob(pytorchjob, JOB_NAMESPACE)
    print(f"List of created {constants.PYTORCHJOB_KIND}s")
    print(TRAINING_CLIENT.list_pytorchjobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.PYTORCHJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_pytorchjob(JOB_NAME, JOB_NAMESPACE)
