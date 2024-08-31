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
import json
import logging
import pytest
import subprocess
from typing import Optional

from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container
from kubernetes.client import V1ResourceRequirements

from kubeflow.storage_initializer.hugging_face import HuggingFaceDatasetParams
from kubeflow.storage_initializer.hugging_face import HuggingFaceModelParams
from kubeflow.storage_initializer.hugging_face import HuggingFaceTrainerParams

from kubeflow.training import TrainingClient
from kubeflow.training import KubeflowOrgV1ReplicaSpec
from kubeflow.training import KubeflowOrgV1PyTorchJob
from kubeflow.training import KubeflowOrgV1PyTorchJobSpec
from kubeflow.training import KubeflowOrgV1RunPolicy
from kubeflow.training import KubeflowOrgV1SchedulingPolicy
from kubeflow.training import constants

from peft import LoraConfig
import transformers

import test.e2e.utils as utils
from test.e2e.constants import TEST_GANG_SCHEDULER_NAME_ENV_KEY
from test.e2e.constants import GANG_SCHEDULERS, NONE_GANG_SCHEDULERS

logging.basicConfig(format="%(message)s")
logging.getLogger("kubeflow.training.api.training_client").setLevel(logging.DEBUG)

TRAINING_CLIENT = TrainingClient(job_kind=constants.PYTORCHJOB_KIND)
CONTAINER_NAME = "pytorch"
GANG_SCHEDULER_NAME = os.getenv(TEST_GANG_SCHEDULER_NAME_ENV_KEY, "")


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in NONE_GANG_SCHEDULERS,
    reason="For gang-scheduling",
)
def test_sdk_e2e_with_gang_scheduling(job_namespace):
    JOB_NAME = "pytorchjob-gang-scheduling"
    container = generate_container()

    master = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(
                scheduler_name=utils.get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
                containers=[container],
            ),
        ),
    )

    worker = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(
                scheduler_name=utils.get_pod_spec_scheduler_name(GANG_SCHEDULER_NAME),
                containers=[container],
            ),
        ),
    )

    unschedulable_pytorchjob = generate_pytorchjob(
        job_namespace,
        JOB_NAME,
        master,
        worker,
        KubeflowOrgV1SchedulingPolicy(min_available=10),
    )
    schedulable_pytorchjob = generate_pytorchjob(
        job_namespace,
        JOB_NAME,
        master,
        worker,
        KubeflowOrgV1SchedulingPolicy(min_available=2),
    )

    TRAINING_CLIENT.create_job(job=unschedulable_pytorchjob, namespace=job_namespace)
    logging.info(f"List of created {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_unschedulable_job_e2e(TRAINING_CLIENT, JOB_NAME, job_namespace)
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"PyTorchJob E2E fails. Exception: {e}")

    TRAINING_CLIENT.update_job(schedulable_pytorchjob, JOB_NAME, job_namespace)
    logging.info(f"List of updated {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_job_e2e(TRAINING_CLIENT, JOB_NAME, job_namespace, wait_timeout=900)
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"PyTorchJob E2E fails. Exception: {e}")

    utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
    TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in GANG_SCHEDULERS,
    reason="For plain scheduling",
)
def test_sdk_e2e(job_namespace):
    JOB_NAME = "pytorchjob-e2e"
    container = generate_container()

    master = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(containers=[container]),
        ),
    )

    worker = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(containers=[container]),
        ),
    )

    pytorchjob = generate_pytorchjob(job_namespace, JOB_NAME, master, worker)

    TRAINING_CLIENT.create_job(job=pytorchjob, namespace=job_namespace)
    logging.info(f"List of created {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_job_e2e(TRAINING_CLIENT, JOB_NAME, job_namespace, wait_timeout=900)
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"PyTorchJob E2E fails. Exception: {e}")

    utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
    TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in GANG_SCHEDULERS,
    reason="For plain scheduling",
)
def test_sdk_e2e_create_from_func(job_namespace):
    JOB_NAME = "pytorchjob-from-func"

    def train_func():
        import time

        for i in range(10):
            print(f"Start training for Epoch {i}")
            time.sleep(1)

    num_workers = 3

    TRAINING_CLIENT.create_job(
        name=JOB_NAME,
        namespace=job_namespace,
        train_func=train_func,
        num_workers=num_workers,
    )

    logging.info(f"List of created {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_job_e2e(TRAINING_CLIENT, JOB_NAME, job_namespace, wait_timeout=900)
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"PyTorchJob create from function E2E fails. Exception: {e}")

    # Verify that PyTorchJob has correct pods.
    pod_names = TRAINING_CLIENT.get_job_pod_names(
        name=JOB_NAME, namespace=job_namespace
    )

    if len(pod_names) != num_workers or f"{JOB_NAME}-worker-0" not in pod_names:
        raise Exception(f"PyTorchJob has incorrect pods: {pod_names}")

    utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
    TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in GANG_SCHEDULERS,
    reason="For plain scheduling",
)
def test_sdk_e2e_create_from_image(job_namespace):
    JOB_NAME = "pytorchjob-from-image"

    TRAINING_CLIENT.create_job(
        name=JOB_NAME,
        namespace=job_namespace,
        base_image="docker.io/hello-world",
        num_workers=1,
    )

    logging.info(f"List of created {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_job_e2e(TRAINING_CLIENT, JOB_NAME, job_namespace, wait_timeout=900)
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"PyTorchJob create from function E2E fails. Exception: {e}")

    utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
    TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)


@pytest.mark.skipif(
    GANG_SCHEDULER_NAME in GANG_SCHEDULERS,
    reason="For plain scheduling",
)
def test_sdk_e2e_create_from_train_api(job_namespace):
    JOB_NAME = "pytorchjob-from-train-api"

    # Use test case from fine-tuning API tutorial.
    # https://www.kubeflow.org/docs/components/training/user-guides/fine-tuning/
    TRAINING_CLIENT.train(
        name=JOB_NAME,
        namespace=job_namespace,
        # BERT model URI and type of Transformer to train it.
        model_provider_parameters=HuggingFaceModelParams(
            model_uri="hf://google-bert/bert-base-cased",
            transformer_type=transformers.AutoModelForSequenceClassification,
            num_labels=5,
        ),
        # In order to save test time, use 8 samples from Yelp dataset.
        dataset_provider_parameters=HuggingFaceDatasetParams(
            repo_id="yelp_review_full",
            split="train[:8]",
        ),
        # Specify HuggingFace Trainer parameters.
        trainer_parameters=HuggingFaceTrainerParams(
            training_parameters=transformers.TrainingArguments(
                output_dir="test_trainer",
                save_strategy="no",
                evaluation_strategy="no",
                do_eval=False,
                disable_tqdm=True,
                log_level="info",
                num_train_epochs=1,
            ),
            # Set LoRA config to reduce number of trainable model parameters.
            lora_config=LoraConfig(
                r=8,
                lora_alpha=8,
                lora_dropout=0.1,
                bias="none",
            ),
        ),
        num_workers=1,  # nodes parameter for torchrun command.
        num_procs_per_worker=1,  # nproc-per-node parameter for torchrun command.
        resources_per_worker={
            "gpu": 0,
            "cpu": 2,
            "memory": "10G",
        },
        storage_config={
            "size": "10Gi",
            "access_modes": ["ReadWriteOnce"],
        },
    )

    logging.info(f"List of created {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_job_e2e(
            TRAINING_CLIENT, JOB_NAME, job_namespace, wait_timeout=300
        )
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"PyTorchJob create from function E2E fails. Exception: {e}")

    utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
    TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)


def generate_pytorchjob(
    job_namespace: str,
    job_name: str,
    master: KubeflowOrgV1ReplicaSpec,
    worker: KubeflowOrgV1ReplicaSpec,
    scheduling_policy: Optional[KubeflowOrgV1SchedulingPolicy] = None,
) -> KubeflowOrgV1PyTorchJob:
    return KubeflowOrgV1PyTorchJob(
        api_version=constants.API_VERSION,
        kind=constants.PYTORCHJOB_KIND,
        metadata=V1ObjectMeta(name=job_name, namespace=job_namespace),
        spec=KubeflowOrgV1PyTorchJobSpec(
            run_policy=KubeflowOrgV1RunPolicy(
                clean_pod_policy="None",
                scheduling_policy=scheduling_policy,
            ),
            pytorch_replica_specs={"Master": master, "Worker": worker},
        ),
    )


def generate_container() -> V1Container:
    return V1Container(
        name=CONTAINER_NAME,
        image="kubeflow/pytorch-dist-mnist:latest",
        args=["--backend", "gloo", "--epochs", "1"],
        resources=V1ResourceRequirements(limits={"memory": "2Gi", "cpu": "0.8"}),
    )


@pytest.fixture(scope="function", autouse=True)
def clean_up_resources():
    # This code runs after each test function
    yield

    try:
        # Display disk usage before cleanup
        print("Disk usage before removing unnecessary files:")
        subprocess.run(["df", "-hT"], check=True)

        # Display Docker disk usage before cleanup
        print("Docker disk usage before removing unnecessary files:")
        subprocess.run(["docker", "system", "df", "-v"], check=True)

        # Display Docker images before cleanup
        print("Docker images before removing unnecessary files:")
        subprocess.run(["docker", "images"], check=True)

        # Display Docker containers before cleanup
        print("Docker containers before removing unnecessary files:")
        subprocess.run(["docker", "ps", "-s", "--all"], check=True)

        # Display Docker volumes before cleanup
        print("Docker volumes before removing unnecessary files:")
        subprocess.run(["docker", "volume", "ls"], check=True)

        # Remove unused Docker volumes
        print("Remove unused Docker volumes:")
        subprocess.run(["docker", "volume", "prune", "-f"], check=True)

        # Additionally list volumes and remove large unused ones
        print("Listing Docker volumes to check for large unused ones:")
        result = subprocess.run(["docker", "volume", "ls", "-q"], check=True, stdout=subprocess.PIPE)
        volumes = result.stdout.decode().splitlines()

        for volume in volumes:
            inspect_result = subprocess.run(["docker", "volume", "inspect", volume], check=True, stdout=subprocess.PIPE)
            volume_details = json.loads(inspect_result.stdout.decode())
            mountpoint = volume_details[0]["Mountpoint"]

            # Check if the mountpoint exists before accessing it
            try:
                volume_size = subprocess.run(["sudo", "du", "-sh", mountpoint], check=True, stdout=subprocess.PIPE).stdout.decode().split()[0]
                print(f"Volume {volume} size: {volume_size}")

                # Example: Remove if larger than 10GB
                size_value = float(volume_size[:-1])
                size_unit = volume_size[-1].upper()

                if size_unit == 'G' and size_value > 10:  # Adjust this condition as needed
                    print(f"Removing volume: {volume}")
                    subprocess.run(["docker", "volume", "rm", volume], check=True)
            except subprocess.CalledProcessError:
                print(f"Volume {volume} not found at expected mountpoint {mountpoint} or cannot access.")
        
        print("Disk usage after removing unnecessary files:")
        subprocess.run(["df", "-hT"], check=True)

        # Display Docker disk usage after cleanup
        print("Docker disk usage after removing unnecessary files:")
        subprocess.run(["docker", "system", "df", "-v"], check=True)

    except subprocess.CalledProcessError as e:
        print(f"Error during Docker cleanup: {e}")
