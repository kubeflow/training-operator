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
from typing import Tuple, Optional

from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container
from kubernetes.client import V1ContainerPort
from kubernetes.client import V1ResourceRequirements

from kubeflow.training import TrainingClient
from kubeflow.training import KubeflowOrgV1ReplicaSpec
from kubeflow.training import KubeflowOrgV1MXJob
from kubeflow.training import KubeflowOrgV1MXJobSpec
from kubeflow.training import KubeflowOrgV1RunPolicy
from kubeflow.training import KubeflowOrgV1SchedulingPolicy
from kubeflow.training.constants import constants

import test.e2e.utils as utils
from test.e2e.constants import TEST_GANG_SCHEDULER_NAME_ENV_KEY
from test.e2e.constants import GANG_SCHEDULERS, NONE_GANG_SCHEDULERS

logging.basicConfig(format="%(message)s")
logging.getLogger("kubeflow.training.api.training_client").setLevel(logging.DEBUG)

TRAINING_CLIENT = TrainingClient(job_kind=constants.MXJOB_KIND)
JOB_NAME = "mxjob-mnist-ci-test"
CONTAINER_NAME = "mxnet"
GANG_SCHEDULER_NAME = os.getenv(TEST_GANG_SCHEDULER_NAME_ENV_KEY, "")


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in NONE_GANG_SCHEDULERS,
    reason="For gang-scheduling",
)
def test_sdk_e2e_with_gang_scheduling(job_namespace):
    worker_container, server_container, scheduler_container = generate_containers()

    worker = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(
                containers=[worker_container],
                scheduler_name=utils.get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
            ),
        ),
    )

    server = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(
                containers=[server_container],
                scheduler_name=utils.get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
            ),
        ),
    )

    scheduler = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(
                containers=[scheduler_container],
                scheduler_name=utils.get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
            ),
        ),
    )

    unschedulable_mxjob = generate_mxjob(
        job_namespace,
        scheduler,
        server,
        worker,
        KubeflowOrgV1SchedulingPolicy(min_available=10),
    )
    schedulable_mxjob = generate_mxjob(
        job_namespace,
        scheduler,
        server,
        worker,
        KubeflowOrgV1SchedulingPolicy(min_available=3),
    )

    TRAINING_CLIENT.create_job(job=unschedulable_mxjob, namespace=job_namespace)
    logging.info(f"List of created {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_unschedulable_job_e2e(TRAINING_CLIENT, JOB_NAME, job_namespace)
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"MXJob E2E fails. Exception: {e}")

    TRAINING_CLIENT.update_job(schedulable_mxjob, JOB_NAME, job_namespace)
    logging.info(f"List of updated {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_job_e2e(TRAINING_CLIENT, JOB_NAME, job_namespace, wait_timeout=900)
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"MXJob E2E fails. Exception: {e}")

    utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
    TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in GANG_SCHEDULERS,
    reason="For plain scheduling",
)
def test_sdk_e2e(job_namespace):
    worker_container, server_container, scheduler_container = generate_containers()

    worker = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(containers=[worker_container]),
        ),
    )

    server = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(containers=[server_container]),
        ),
    )

    scheduler = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(containers=[scheduler_container]),
        ),
    )

    mxjob = generate_mxjob(job_namespace, scheduler, server, worker)

    TRAINING_CLIENT.create_job(job=mxjob, namespace=job_namespace)
    logging.info(f"List of created {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_job_e2e(TRAINING_CLIENT, JOB_NAME, job_namespace, wait_timeout=900)
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"MXJob E2E fails. Exception: {e}")

    utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
    TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)


def generate_mxjob(
    job_namespace: str,
    scheduler: KubeflowOrgV1ReplicaSpec,
    server: KubeflowOrgV1ReplicaSpec,
    worker: KubeflowOrgV1ReplicaSpec,
    scheduling_policy: Optional[KubeflowOrgV1SchedulingPolicy] = None,
) -> KubeflowOrgV1MXJob:
    return KubeflowOrgV1MXJob(
        api_version=constants.API_VERSION,
        kind=constants.MXJOB_KIND,
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=job_namespace),
        spec=KubeflowOrgV1MXJobSpec(
            job_mode="MXTrain",
            run_policy=KubeflowOrgV1RunPolicy(
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
        image="docker.io/kubeflow/mxnet-gpu:latest",
        command=["/usr/local/bin/python3"],
        args=[
            "/mxnet/mxnet/example/image-classification/train_mnist.py",
            "--num-epochs",
            "1",
            "--num-examples",
            "1000",
            "--kv-store",
            "dist_sync",
        ],
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
        resources=V1ResourceRequirements(limits={"memory": "2Gi", "cpu": "0.8"}),
    )

    server_container = V1Container(
        name=CONTAINER_NAME,
        image="docker.io/kubeflow/mxnet-gpu:latest",
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
        resources=V1ResourceRequirements(limits={"memory": "1Gi", "cpu": "0.4"}),
    )

    scheduler_container = V1Container(
        name=CONTAINER_NAME,
        image="docker.io/kubeflow/mxnet-gpu:latest",
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")],
        resources=V1ResourceRequirements(limits={"memory": "1Gi", "cpu": "0.4"}),
    )

    return worker_container, server_container, scheduler_container
