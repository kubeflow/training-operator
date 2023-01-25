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

from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container

from kubeflow.training import TrainingClient
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1PyTorchJob
from kubeflow.training import KubeflowOrgV1PyTorchJobSpec
from kubeflow.training import V1RunPolicy
from kubeflow.training import V1SchedulingPolicy
from kubeflow.training.constants import constants

from test.e2e.utils import verify_job_e2e, verify_unschedulable_job_e2e, get_pod_spec_scheduler_name
from test.e2e.constants import TEST_GANG_SCHEDULER_NAME_ENV_KEY
from test.e2e.constants import GANG_SCHEDULERS, NONE_GANG_SCHEDULERS

logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "pytorchjob-mnist-ci-test"
JOB_NAMESPACE = "default"
CONTAINER_NAME = "pytorch"
GANG_SCHEDULER_NAME = os.getenv(TEST_GANG_SCHEDULER_NAME_ENV_KEY)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in NONE_GANG_SCHEDULERS, reason="For gang-scheduling",
)
def test_sdk_e2e_with_gang_scheduling():
    container = generate_container()

    master = V1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(spec=V1PodSpec(
            scheduler_name=get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
            containers=[container],
        )),
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(spec=V1PodSpec(
            scheduler_name=get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
            containers=[container],
        )),
    )

    unschedulable_pytorchjob = generate_pytorchjob(master, worker, V1SchedulingPolicy(min_available=10))
    schedulable_pytorchjob = generate_pytorchjob(master, worker, V1SchedulingPolicy(min_available=2))

    TRAINING_CLIENT.create_pytorchjob(unschedulable_pytorchjob, JOB_NAMESPACE)
    logging.info(f"List of created {constants.PYTORCHJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_pytorchjobs(JOB_NAMESPACE))

    verify_unschedulable_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.PYTORCHJOB_KIND,
    )

    TRAINING_CLIENT.patch_pytorchjob(schedulable_pytorchjob, JOB_NAME, JOB_NAMESPACE)
    logging.info(f"List of patched {constants.PYTORCHJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_pytorchjobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.PYTORCHJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_pytorchjob(JOB_NAME, JOB_NAMESPACE)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in GANG_SCHEDULERS, reason="For plain scheduling",
)
def test_sdk_e2e():
    container = generate_container()

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

    pytorchjob = generate_pytorchjob(master, worker)

    TRAINING_CLIENT.create_pytorchjob(pytorchjob, JOB_NAMESPACE)
    logging.info(f"List of created {constants.PYTORCHJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_pytorchjobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.PYTORCHJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_pytorchjob(JOB_NAME, JOB_NAMESPACE)


def generate_pytorchjob(
    master: V1ReplicaSpec,
    worker: V1ReplicaSpec,
    scheduling_policy: V1SchedulingPolicy = None,
) -> KubeflowOrgV1PyTorchJob:
    return KubeflowOrgV1PyTorchJob(
        api_version="kubeflow.org/v1",
        kind="PyTorchJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1PyTorchJobSpec(
            run_policy=V1RunPolicy(
                clean_pod_policy="None",
                scheduling_policy=scheduling_policy,
            ),
            pytorch_replica_specs={"Master": master, "Worker": worker},
        ),
    )


def generate_container() -> V1Container:
    return V1Container(
        name=CONTAINER_NAME,
        image="gcr.io/kubeflow-ci/pytorch-dist-mnist-test:v1.0",
        args=["--backend", "gloo"],
    )
