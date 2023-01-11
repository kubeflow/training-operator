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
from kubeflow.training import KubeflowOrgV1MPIJob
from kubeflow.training import KubeflowOrgV1MPIJobSpec
from kubeflow.training import V1RunPolicy
from kubeflow.training.constants import constants

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
SDK_TEST_NAMESPACE = "default"
JOB_NAME = "mpijob-mxnet-ci-test"
CONTAINER_NAME = "mpi"


def test_sdk_e2e():
    master_container = V1Container(
        name=CONTAINER_NAME,
        image="horovod/horovod:0.20.0-tf2.3.0-torch1.6.0-mxnet1.5.0-py3.7-cpu",
        command=["mpirun"],
        args=[
            "-np",
            "1",
            "--allow-run-as-root",
            "-bind-to",
            "none",
            "-map-by",
            "slot",
            "-x",
            "LD_LIBRARY_PATH",
            "-x",
            "PATH",
            "-mca",
            "pml",
            "ob1",
            "-mca",
            "btl",
            "^openib",
            # "python", "/examples/tensorflow2_mnist.py"]
            "python",
            "/examples/pytorch_mnist.py",
            "--epochs",
            "1",
        ],
    )

    worker_container = V1Container(
        name="mpi",
        image="horovod/horovod:0.20.0-tf2.3.0-torch1.6.0-mxnet1.5.0-py3.7-cpu",
    )

    master = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[master_container])),
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[worker_container])),
    )

    mpijob = KubeflowOrgV1MPIJob(
        api_version="kubeflow.org/v1",
        kind="MPIJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=SDK_TEST_NAMESPACE),
        spec=KubeflowOrgV1MPIJobSpec(
            slots_per_worker=1,
            run_policy=V1RunPolicy(clean_pod_policy="None",),
            mpi_replica_specs={"Launcher": master, "Worker": worker},
        ),
    )

    TRAINING_CLIENT.create_mpijob(mpijob, SDK_TEST_NAMESPACE)

    TRAINING_CLIENT.wait_for_job_conditions(
        JOB_NAME, SDK_TEST_NAMESPACE, constants.MPIJOB_KIND
    )

    TRAINING_CLIENT.get_job_logs(JOB_NAME, SDK_TEST_NAMESPACE, container=CONTAINER_NAME)

    TRAINING_CLIENT.delete_mpijob(JOB_NAME, SDK_TEST_NAMESPACE)
