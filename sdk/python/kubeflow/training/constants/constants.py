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
from typing import Union

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

# TFJob constants.
TFJOB_KIND = "TFJob"
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
PYTORCHJOB_PLURAL = "pytorchjobs"
PYTORCHJOB_CONTAINER = "pytorch"
PYTORCHJOB_REPLICA_TYPES = (REPLICA_TYPE_MASTER.lower(), REPLICA_TYPE_WORKER.lower())

PYTORCHJOB_BASE_IMAGE = "docker.io/pytorch/pytorch:1.12.1-cuda11.3-cudnn8-runtime"

# MXJob constants
MXJOB_KIND = "MXJob"
MXJOB_PLURAL = "mxjobs"
MXJOB_CONTAINER = "mxnet"
MXJOB_REPLICA_TYPES = (
    REPLICA_TYPE_SCHEDULER.lower(),
    REPLICA_TYPE_SERVER.lower(),
    REPLICA_TYPE_WORKER.lower(),
)

# XGBoostJob constants
XGBOOSTJOB_KIND = "XGBoostJob"
XGBOOSTJOB_PLURAL = "xgboostjobs"
XGBOOSTJOB_CONTAINER = "xgboost"
XGBOOSTJOB_REPLICA_TYPES = (REPLICA_TYPE_MASTER.lower(), REPLICA_TYPE_WORKER.lower())

# MPIJob constants
MPIJOB_KIND = "MPIJob"
MPIJOB_PLURAL = "mpijobs"
MPIJOB_CONTAINER = "mpi"
MPIJOB_REPLICA_TYPES = (REPLICA_TYPE_LAUNCHER.lower(), REPLICA_TYPE_WORKER.lower())

# PaddleJob constants
PADDLEJOB_KIND = "PaddleJob"
PADDLEJOB_PLURAL = "paddlejobs"
PADDLEJOB_CONTAINER = "paddle"
PADDLEJOB_REPLICA_TYPES = (REPLICA_TYPE_MASTER.lower(), REPLICA_TYPE_WORKER.lower())

PADDLEJOB_BASE_IMAGE = (
    "docker.io/paddlepaddle/paddle:2.4.0rc0-gpu-cuda11.2-cudnn8.1-trt8.0"
)


# Dictionary to get plural, model, and container for each Job kind.
JOB_PARAMETERS = {
    TFJOB_KIND: {
        "model": models.KubeflowOrgV1TFJob,
        "plural": TFJOB_PLURAL,
        "container": TFJOB_CONTAINER,
        "base_image": TFJOB_BASE_IMAGE,
    },
    PYTORCHJOB_KIND: {
        "model": models.KubeflowOrgV1PyTorchJob,
        "plural": PYTORCHJOB_PLURAL,
        "container": PYTORCHJOB_CONTAINER,
        "base_image": PYTORCHJOB_BASE_IMAGE,
    },
    MXJOB_KIND: {
        "model": models.KubeflowOrgV1MXJob,
        "plural": MXJOB_PLURAL,
        "container": MXJOB_CONTAINER,
    },
    XGBOOSTJOB_KIND: {
        "model": models.KubeflowOrgV1XGBoostJob,
        "plural": XGBOOSTJOB_PLURAL,
        "container": XGBOOSTJOB_CONTAINER,
    },
    MPIJOB_KIND: {
        "model": models.KubeflowOrgV1MPIJob,
        "plural": MPIJOB_PLURAL,
        "container": MPIJOB_CONTAINER,
    },
    PADDLEJOB_KIND: {
        "model": models.KubeflowOrgV1PaddleJob,
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
