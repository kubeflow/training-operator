# Copyright 2024 kubeflow.org.
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

import logging
import test.e2e.utils as utils
 
from kubeflow.storage_initializer.hugging_face import HuggingFaceDatasetParams
from kubeflow.storage_initializer.hugging_face import HuggingFaceModelParams
from kubeflow.storage_initializer.hugging_face import HuggingFaceTrainerParams
from kubeflow.training import constants
from kubeflow.training import TrainingClient
from kubernetes import client
from kubernetes import config
from peft import LoraConfig
import transformers

logging.basicConfig(format="%(message)s")
logging.getLogger("kubeflow.training.api.training_client").setLevel(logging.DEBUG)

TRAINING_CLIENT = TrainingClient(job_kind=constants.PYTORCHJOB_KIND)
JOB_NAME = "test-train-api"


def test_train_api(job_namespace):
    num_workers = 4

    # Use test case from fine-tuning API tutorial
    # https://www.kubeflow.org/docs/components/training/user-guides/fine-tuning/
    TRAINING_CLIENT.train(
        name=JOB_NAME,
        namespace=job_namespace,
        # BERT model URI and type of Transformer to train it.
        model_provider_parameters=HuggingFaceModelParams(
            model_uri="hf://google-bert/bert-base-cased",
            transformer_type=transformers.AutoModelForSequenceClassification,
        ),
        # In order to save test time, use 8 samples from Yelp dataset.
        dataset_provider_parameters=HuggingFaceDatasetParams(
            repo_id="yelp_review_full",
            split="train[:8]",
        ),
        # Specify HuggingFace Trainer parameters. In this example, we will skip evaluation and model checkpoints.
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
        num_workers=num_workers,  # nodes parameter for torchrun command.
        num_procs_per_worker=2,  # nproc-per-node parameter for torchrun command.
        resources_per_worker={
            "gpu": 2,
            "cpu": 5,
            "memory": "10G",
        },
    )

    logging.info(f"List of created {TRAINING_CLIENT.job_kind}s")
    logging.info(TRAINING_CLIENT.list_jobs(job_namespace))

    try:
        utils.verify_job_e2e(
            TRAINING_CLIENT, JOB_NAME, job_namespace, wait_timeout=60 * 60
        )
        logging.info(f"Training job {JOB_NAME} is succeded.")
    except Exception as e:
        utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)
        TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)
        raise Exception(f"Training job {JOB_NAME} is failed. Exception: {e}")

    # Verify that training job has correct pods.
    pod_names = TRAINING_CLIENT.get_job_pod_names(
        name=JOB_NAME, namespace=job_namespace
    )

    # if len(pod_names) != num_workers or f"{JOB_NAME}-worker-0" not in pod_names:
    if len(pod_names) != num_workers:
        raise Exception(f"Training job has incorrect pods: {pod_names}")

    utils.print_job_results(TRAINING_CLIENT, JOB_NAME, job_namespace)

    # Get and print the logs of the master pod
    master_pod_name = next((name for name in pod_names if "master" in name), None)
    if master_pod_name:
        config.load_kube_config()  # Load kube config to interact with the cluster
        v1 = client.CoreV1Api()
        try:
            pod_logs = v1.read_namespaced_pod_log(
                name=master_pod_name, namespace=job_namespace
            )
            logging.info(f"Logs of master pod {master_pod_name}:\n{pod_logs}")
        except client.exceptions.ApiException as e:
            logging.error(f"Failed to get logs for pod {master_pod_name}: {e}")

    TRAINING_CLIENT.delete_job(JOB_NAME, job_namespace)


if __name__ == "__main__":
    test_train_api(job_namespace="default")
