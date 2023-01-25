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
import pytest
from typing import Tuple

from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container

from kubeflow.training import TrainingClient
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1MPIJob
from kubeflow.training import KubeflowOrgV1MPIJobSpec
from kubeflow.training import V1RunPolicy
from kubeflow.training import V1SchedulingPolicy
from kubeflow.training.constants import constants

from test.e2e.utils import verify_job_e2e, verify_unschedulable_job_e2e, get_pod_spec_scheduler_name
from test.e2e.constants import TEST_GANG_SCHEDULER_NAME_ENV_KEY
from test.e2e.constants import GANG_SCHEDULERS, NONE_GANG_SCHEDULERS

logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "mpijob-mxnet-ci-test"
JOB_NAMESPACE = "default"
CONTAINER_NAME = "mpi"
GANG_SCHEDULER_NAME = os.getenv(TEST_GANG_SCHEDULER_NAME_ENV_KEY)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in NONE_GANG_SCHEDULERS, reason="For gang-scheduling",
)
def test_sdk_e2e_with_gang_scheduling():
    launcher_container, worker_container = generate_containers()

    launcher = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(
            containers=[launcher_container],
            scheduler_name=get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
        )),
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(
            containers=[worker_container],
            scheduler_name=get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
        )),
    )

    mpijob = generate_mpijob(launcher, worker, V1SchedulingPolicy(min_available=10))
    patched_mpijob = generate_mpijob(launcher, worker, V1SchedulingPolicy(min_available=2))

    TRAINING_CLIENT.create_mpijob(mpijob, JOB_NAMESPACE)
    logging.info(f"List of created {constants.MPIJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_mpijobs(JOB_NAMESPACE))

    verify_unschedulable_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.MPIJOB_KIND,
    )

    TRAINING_CLIENT.patch_mpijob(patched_mpijob, JOB_NAME, JOB_NAMESPACE)
    logging.info(f"List of patched {constants.MPIJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_mpijobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.MPIJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_mpijob(JOB_NAME, JOB_NAMESPACE)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in GANG_SCHEDULERS, reason="For plain scheduling",
)
def test_sdk_e2e():
    launcher_container, worker_container = generate_containers()

    launcher = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[launcher_container])),
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[worker_container])),
    )

    mpijob = generate_mpijob(launcher, worker, None)

    TRAINING_CLIENT.create_mpijob(mpijob, JOB_NAMESPACE)
    logging.info(f"List of created {constants.MPIJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_mpijobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.MPIJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_mpijob(JOB_NAME, JOB_NAMESPACE)


def generate_mpijob(
    launcher: V1ReplicaSpec,
    worker: V1ReplicaSpec,
    scheduling_policy: V1SchedulingPolicy = None,
) -> KubeflowOrgV1MPIJob:
    return KubeflowOrgV1MPIJob(
        api_version="kubeflow.org/v1",
        kind="MPIJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1MPIJobSpec(
            slots_per_worker=1,
            run_policy=V1RunPolicy(
                clean_pod_policy="None",
                scheduling_policy=scheduling_policy,
            ),
            mpi_replica_specs={"Launcher": launcher, "Worker": worker},
        ),
    )


def generate_containers() -> Tuple[V1Container, V1Container]:
    launcher_container = V1Container(
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

    return launcher_container, worker_container
