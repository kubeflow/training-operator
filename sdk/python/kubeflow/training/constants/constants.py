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

from kubeflow.training import models
from typing import Union, Dict
from kubeflow.storage_initializer.constants import INIT_CONTAINER_MOUNT_PATH

# How long to wait in seconds for requests to the Kubernetes API Server.
DEFAULT_TIMEOUT = 120

# The default PIP index URL to download Python packages.
DEFAULT_PIP_INDEX_URL = "https://pypi.org/simple"

# Annotation to disable Istio sidecar.
ISTIO_SIDECAR_INJECTION = "sidecar.istio.io/inject"

# Common constants.
GROUP = "kubeflow.org"
VERSION = "v1"
API_VERSION = f"{GROUP}/{VERSION}"

# Kind for pod.
POD_KIND = "Pod"

# Pending status for pod phase.
POD_PHASE_PENDING = "Pending"


# Training Job conditions.
JOB_CONDITION_CREATED = "Created"
JOB_CONDITION_RUNNING = "Running"
JOB_CONDITION_RESTARTING = "Restarting"
JOB_CONDITION_SUCCEEDED = "Succeeded"
JOB_CONDITION_FAILED = "Failed"
JOB_CONDITIONS = {
    JOB_CONDITION_CREATED,
    JOB_CONDITION_RUNNING,
    JOB_CONDITION_RESTARTING,
    JOB_CONDITION_SUCCEEDED,
    JOB_CONDITION_FAILED,
}
# True means that Training Job is in this condition.
CONDITION_STATUS_TRUE = "True"

# Job Label Names
JOB_NAME_LABEL = "training.kubeflow.org/job-name"
JOB_ROLE_LABEL = "training.kubeflow.org/job-role"
JOB_ROLE_MASTER = "master"
REPLICA_TYPE_LABEL = "training.kubeflow.org/replica-type"
REPLICA_INDEX_LABEL = "training.kubeflow.org/replica-index"

# Various replica types.
REPLICA_TYPE_CHIEF = "Chief"
REPLICA_TYPE_PS = "PS"
REPLICA_TYPE_MASTER = "Master"
REPLICA_TYPE_WORKER = "Worker"
REPLICA_TYPE_SCHEDULER = "Scheduler"
REPLICA_TYPE_SERVER = "Server"
REPLICA_TYPE_LAUNCHER = "Launcher"

# Constants for Train API.
STORAGE_INITIALIZER = "storage-initializer"
# The default value for dataset and model storage PVC.
PVC_DEFAULT_SIZE = "10Gi"
# The default value for PVC access modes.
PVC_DEFAULT_ACCESS_MODES = ["ReadWriteOnce", "ReadOnlyMany"]


# TODO (andreyvelich): We should add image tag for Storage Initializer and Trainer.
STORAGE_INITIALIZER_IMAGE = "docker.io/kubeflow/storage-initializer"

STORAGE_INITIALIZER_VOLUME_MOUNT = models.V1VolumeMount(
    name=STORAGE_INITIALIZER,
    mount_path=INIT_CONTAINER_MOUNT_PATH,
)
STORAGE_INITIALIZER_VOLUME = models.V1Volume(
    name=STORAGE_INITIALIZER,
    persistent_volume_claim=models.V1PersistentVolumeClaimVolumeSource(
        claim_name=STORAGE_INITIALIZER
    ),
)
TRAINER_TRANSFORMER_IMAGE = "docker.io/kubeflow/trainer-huggingface"

# TFJob constants.
TFJOB_KIND = "TFJob"
TFJOB_MODEL = "KubeflowOrgV1TFJob"
TFJOB_PLURAL = "tfjobs"
TFJOB_CONTAINER = "tensorflow"
TFJOB_REPLICA_TYPES = (
    REPLICA_TYPE_PS.lower(),
    REPLICA_TYPE_CHIEF.lower(),
    REPLICA_TYPE_WORKER.lower(),
)

TFJOB_BASE_IMAGE = "docker.io/tensorflow/tensorflow:2.9.1"
TFJOB_BASE_IMAGE_GPU = "docker.io/tensorflow/tensorflow:2.9.1-gpu"

# PyTorchJob constants
PYTORCHJOB_KIND = "PyTorchJob"
PYTORCHJOB_MODEL = "KubeflowOrgV1PyTorchJob"
PYTORCHJOB_PLURAL = "pytorchjobs"
PYTORCHJOB_CONTAINER = "pytorch"
PYTORCHJOB_REPLICA_TYPES = (REPLICA_TYPE_MASTER.lower(), REPLICA_TYPE_WORKER.lower())
PYTORCHJOB_BASE_IMAGE = "docker.io/pytorch/pytorch:2.1.2-cuda11.8-cudnn8-runtime"

# MXJob constants
MXJOB_KIND = "MXJob"
MXJOB_MODEL = "KubeflowOrgV1MXJob"
MXJOB_PLURAL = "mxjobs"
MXJOB_CONTAINER = "mxnet"
MXJOB_REPLICA_TYPES = (
    REPLICA_TYPE_SCHEDULER.lower(),
    REPLICA_TYPE_SERVER.lower(),
    REPLICA_TYPE_WORKER.lower(),
)

# XGBoostJob constants
XGBOOSTJOB_KIND = "XGBoostJob"
XGBOOSTJOB_MODEL = "KubeflowOrgV1XGBoostJob"
XGBOOSTJOB_PLURAL = "xgboostjobs"
XGBOOSTJOB_CONTAINER = "xgboost"
XGBOOSTJOB_REPLICA_TYPES = (REPLICA_TYPE_MASTER.lower(), REPLICA_TYPE_WORKER.lower())

# MPIJob constants
MPIJOB_KIND = "MPIJob"
MPIJOB_MODEL = "KubeflowOrgV1MPIJob"
MPIJOB_PLURAL = "mpijobs"
MPIJOB_CONTAINER = "mpi"
MPIJOB_REPLICA_TYPES = (REPLICA_TYPE_LAUNCHER.lower(), REPLICA_TYPE_WORKER.lower())

# PaddleJob constants
PADDLEJOB_KIND = "PaddleJob"
PADDLEJOB_MODEL = "KubeflowOrgV1PaddleJob"
PADDLEJOB_PLURAL = "paddlejobs"
PADDLEJOB_CONTAINER = "paddle"
PADDLEJOB_REPLICA_TYPES = (REPLICA_TYPE_MASTER.lower(), REPLICA_TYPE_WORKER.lower())

PADDLEJOB_BASE_IMAGE = (
    "docker.io/paddlepaddle/paddle:2.4.0rc0-gpu-cuda11.2-cudnn8.1-trt8.0"
)


# Dictionary to get plural, model, and container for each Job kind.
JOB_PARAMETERS = {
    TFJOB_KIND: {
        "model": TFJOB_MODEL,
        "plural": TFJOB_PLURAL,
        "container": TFJOB_CONTAINER,
        "base_image": TFJOB_BASE_IMAGE,
    },
    PYTORCHJOB_KIND: {
        "model": PYTORCHJOB_MODEL,
        "plural": PYTORCHJOB_PLURAL,
        "container": PYTORCHJOB_CONTAINER,
        "base_image": PYTORCHJOB_BASE_IMAGE,
    },
    MXJOB_KIND: {
        "model": MXJOB_MODEL,
        "plural": MXJOB_PLURAL,
        "container": MXJOB_CONTAINER,
        "base_image": "TODO",
    },
    XGBOOSTJOB_KIND: {
        "model": XGBOOSTJOB_MODEL,
        "plural": XGBOOSTJOB_PLURAL,
        "container": XGBOOSTJOB_CONTAINER,
        "base_image": "TODO",
    },
    MPIJOB_KIND: {
        "model": MPIJOB_MODEL,
        "plural": MPIJOB_PLURAL,
        "container": MPIJOB_CONTAINER,
        "base_image": "TODO",
    },
    PADDLEJOB_KIND: {
        "model": PADDLEJOB_MODEL,
        "plural": PADDLEJOB_PLURAL,
        "container": PADDLEJOB_CONTAINER,
        "base_image": PADDLEJOB_BASE_IMAGE,
    },
}

# Tuple of all Job models.
JOB_MODELS = tuple([d["model"] for d in list(JOB_PARAMETERS.values())])

# Union type of all Job models.
JOB_MODELS_TYPE = Union[
    models.KubeflowOrgV1TFJob,
    models.KubeflowOrgV1PyTorchJob,
    models.KubeflowOrgV1MXJob,
    models.KubeflowOrgV1XGBoostJob,
    models.KubeflowOrgV1MPIJob,
    models.KubeflowOrgV1PaddleJob,
]
