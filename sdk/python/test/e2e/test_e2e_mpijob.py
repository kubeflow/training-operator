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

from test.e2e.utils import verify_job_e2e

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "mpijob-mxnet-ci-test"
JOB_NAMESPACE = "default"
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
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1MPIJobSpec(
            slots_per_worker=1,
            run_policy=V1RunPolicy(clean_pod_policy="None",),
            mpi_replica_specs={"Launcher": master, "Worker": worker},
        ),
    )

    TRAINING_CLIENT.create_mpijob(mpijob, JOB_NAMESPACE)
    print(f"List of created {constants.MPIJOB_KIND}s")
    print(TRAINING_CLIENT.list_mpijobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT, JOB_NAME, JOB_NAMESPACE, constants.MPIJOB_KIND, CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_mpijob(JOB_NAME, JOB_NAMESPACE)
