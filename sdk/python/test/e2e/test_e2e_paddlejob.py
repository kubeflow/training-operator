# Copyright 2022 kubeflow.org.
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
from kubeflow.training import KubeflowOrgV1PaddleJob
from kubeflow.training import KubeflowOrgV1PaddleJobSpec
from kubeflow.training import V1RunPolicy
from kubeflow.training.constants import constants

from test.e2e.utils import verify_job_e2e

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "paddlejob-cpu-ci-test"
JOB_NAMESPACE = "default"
CONTAINER_NAME = "paddle"


def test_sdk_e2e():
    container = V1Container(
        name=CONTAINER_NAME,
        image="docker.io/paddlepaddle/paddle:2.4.0rc0-cpu",
        command=["python"],
        args=["-m", "paddle.distributed.launch", "run_check"],
    )

    worker = V1ReplicaSpec(
        replicas=2,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[container])),
    )

    paddlejob = KubeflowOrgV1PaddleJob(
        api_version="kubeflow.org/v1",
        kind="PaddleJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1PaddleJobSpec(
            run_policy=V1RunPolicy(clean_pod_policy="None",),
            paddle_replica_specs={"Worker": worker},
        ),
    )

    TRAINING_CLIENT.create_paddlejob(paddlejob, JOB_NAMESPACE)
    print(f"List of created {constants.PADDLEJOB_KIND}s")
    print(TRAINING_CLIENT.list_paddlejobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.PADDLEJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_paddlejob(JOB_NAME, JOB_NAMESPACE)
