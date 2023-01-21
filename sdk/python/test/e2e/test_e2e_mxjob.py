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
from kubernetes.client import V1ContainerPort

from kubeflow.training import TrainingClient
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1MXJob
from kubeflow.training import KubeflowOrgV1MXJobSpec
from kubeflow.training import V1RunPolicy
from kubeflow.training import V1SchedulingPolicy
from kubeflow.training.constants import constants

from test.e2e.utils import verify_job_e2e, verify_unschedulable_job_e2e, get_pod_spec_scheduler_name
from test.e2e.constants import TEST_GANG_SCHEDULER_NAME_ENV_KEY
from test.e2e.constants import GANG_SCHEDULERS, NONE_GANG_SCHEDULERS

logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "mxjob-mnist-ci-test"
JOB_NAMESPACE = "default"
CONTAINER_NAME = "mxnet"
GANG_SCHEDULER_NAME = os.getenv(TEST_GANG_SCHEDULER_NAME_ENV_KEY)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in NONE_GANG_SCHEDULERS, reason="For gang-scheduling",
)
def test_sdk_e2e_with_gang_scheduling():
    worker_container, server_container, scheduler_container = generate_containers()

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(
            containers=[worker_container],
            scheduler_name=get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
        )),
    )

    server = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(
            containers=[server_container],
            scheduler_name=get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
        )),
    )

    scheduler = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(
            containers=[scheduler_container],
            scheduler_name=get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
        )),
    )

    unschedulable_mxjob = generate_mxjob(scheduler, server, worker, V1SchedulingPolicy(min_available=10))
    schedulable_mxjob = generate_mxjob(scheduler, server, worker, V1SchedulingPolicy(min_available=3))

    TRAINING_CLIENT.create_mxjob(unschedulable_mxjob, JOB_NAMESPACE)
    logging.info(f"List of created {constants.MXJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_mxjobs(JOB_NAMESPACE))

    verify_unschedulable_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.MXJOB_KIND,
    )

    TRAINING_CLIENT.patch_mxjob(schedulable_mxjob, JOB_NAME, JOB_NAMESPACE)
    logging.info(f"List of patched {constants.MXJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_mxjobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.MXJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_mxjob(JOB_NAME, JOB_NAMESPACE)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in GANG_SCHEDULERS, reason="For plain scheduling",
)
def test_sdk_e2e():
    worker_container, server_container, scheduler_container = generate_containers()

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

    mxjob = generate_mxjob(scheduler, server, worker)

    TRAINING_CLIENT.create_mxjob(mxjob, JOB_NAMESPACE)
    logging.info(f"List of created {constants.MXJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_mxjobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.MXJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_mxjob(JOB_NAME, JOB_NAMESPACE)


def generate_mxjob(
    scheduler: V1ReplicaSpec,
    server: V1ReplicaSpec,
    worker: V1ReplicaSpec,
    scheduling_policy: V1SchedulingPolicy = None,
) -> KubeflowOrgV1MXJob:
    return KubeflowOrgV1MXJob(
        api_version="kubeflow.org/v1",
        kind="MXJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1MXJobSpec(
            job_mode="MXTrain",
            run_policy=V1RunPolicy(
                clean_pod_policy="None",
                scheduling_policy=scheduling_policy,
            ),
            mx_replica_specs={
                "Scheduler": scheduler,
                "Server": server,
                "Worker": worker,
            },
        ),
    )


def generate_containers() -> Tuple[V1Container, V1Container, V1Container]:
    worker_container = V1Container(
        name=CONTAINER_NAME,
        # TODO (tenzen-y): Replace the below image with the kubeflow hosted image
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        command=["/usr/local/bin/python3"],
        args=[
            "incubator-mxnet/example/image-classification/train_mnist.py",
            "--num-epochs",
            "1",
            "--num-examples",
            "1000",
            "--kv-store",
            "dist_sync",
        ],
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
    )

    server_container = V1Container(
        name=CONTAINER_NAME,
        # TODO (tenzen-y): Replace the below image with the kubeflow hosted image
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
    )

    scheduler_container = V1Container(
        name=CONTAINER_NAME,
        # TODO (tenzen-y): Replace the below image with the kubeflow hosted image
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
    )

    return worker_container, server_container, scheduler_container
