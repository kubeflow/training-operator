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
from kubeflow.training import V1RunPolicy
from kubeflow.training import KubeflowOrgV1TFJob
from kubeflow.training import KubeflowOrgV1TFJobSpec
from kubeflow.training.constants import constants

from test.e2e.utils import verify_job_e2e

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "tfjob-mnist-ci-test"
JOB_NAMESPACE = "default"
CONTAINER_NAME = "tensorflow"


def test_sdk_e2e():
    container = V1Container(
        name=CONTAINER_NAME,
        image="gcr.io/kubeflow-ci/tf-mnist-with-summaries:1.0",
        command=[
            "python",
            "/var/tf_mnist/mnist_with_summaries.py",
            "--log_dir=/train/logs",
            "--learning_rate=0.01",
            "--batch_size=150",
        ],
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[container])),
    )

    tfjob = KubeflowOrgV1TFJob(
        api_version="kubeflow.org/v1",
        kind="TFJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1TFJobSpec(
            run_policy=V1RunPolicy(clean_pod_policy="None",),
            tf_replica_specs={"Worker": worker},
        ),
    )

    TRAINING_CLIENT.create_tfjob(tfjob, JOB_NAMESPACE)
    print(f"List of created {constants.TFJOB_KIND}s")
    print(TRAINING_CLIENT.list_tfjobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT, JOB_NAME, JOB_NAMESPACE, constants.TFJOB_KIND, CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_tfjob(JOB_NAME, JOB_NAMESPACE)
