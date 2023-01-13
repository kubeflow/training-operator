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
import logging

from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container
from kubernetes.client import V1ContainerPort

from kubeflow.training import TrainingClient
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1MXJob
from kubeflow.training import KubeflowOrgV1MXJobSpec
from kubeflow.training import V1RunPolicy
from kubeflow.training.constants import constants

from test.e2e.utils import verify_job_e2e

logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "mxjob-mnist-ci-test"
JOB_NAMESPACE = "default"
CONTAINER_NAME = "mxnet"


def test_sdk_e2e():
    worker_container = V1Container(
        name=CONTAINER_NAME,
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        command=["/usr/local/bin/python3"],
        args=[
            "incubator-mxnet/example/image-classification/train_mnist.py",
            "--num-epochs",
            "5",
            "--num-examples",
            "1000",
            "--kv-store",
            "dist_sync",
        ],
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
    )

    server_container = V1Container(
        name=CONTAINER_NAME,
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
    )

    scheduler_container = V1Container(
        name=CONTAINER_NAME,
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[worker_container])),
    )

    server = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[server_container])),
    )

    scheduler = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[scheduler_container])),
    )

    mxjob = KubeflowOrgV1MXJob(
        api_version="kubeflow.org/v1",
        kind="MXJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1MXJobSpec(
            job_mode="MXTrain",
            run_policy=V1RunPolicy(clean_pod_policy="None",),
            mx_replica_specs={
                "Scheduler": scheduler,
                "Server": server,
                "Worker": worker,
            },
        ),
    )

    TRAINING_CLIENT.create_mxjob(mxjob, JOB_NAMESPACE)
    logging.info(f"List of created {constants.MXJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_mxjobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT, JOB_NAME, JOB_NAMESPACE, constants.MXJOB_KIND, CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_mxjob(JOB_NAME, JOB_NAMESPACE)
