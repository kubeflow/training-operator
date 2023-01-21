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

# How long to wait in seconds for requests to the Kubernetes API Server.
DEFAULT_TIMEOUT = 120

# Common constants.
KUBEFLOW_GROUP = "kubeflow.org"
OPERATOR_VERSION = "v1"

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

# TFJob constants.
TFJOB_KIND = "TFJob"
TFJOB_PLURAL = "tfjobs"
TFJOB_CONTAINER = "tensorflow"
TFJOB_REPLICA_TYPES = {"ps", "chief", "worker"}

TFJOB_BASE_IMAGE = "docker.io/tensorflow/tensorflow:2.9.1"
TFJOB_BASE_IMAGE_GPU = "docker.io/tensorflow/tensorflow:2.9.1-gpu"

# PyTorchJob constants
PYTORCHJOB_KIND = "PyTorchJob"
PYTORCHJOB_PLURAL = "pytorchjobs"
PYTORCHJOB_CONTAINER = "pytorch"
PYTORCHJOB_REPLICA_TYPES = {"master", "worker"}

PYTORCHJOB_BASE_IMAGE = "docker.io/pytorch/pytorch:1.12.1-cuda11.3-cudnn8-runtime"

# MXJob constants
MXJOB_KIND = "MXJob"
MXJOB_PLURAL = "mxjobs"
MXJOB_REPLICA_TYPES = {"scheduler", "server", "worker"}

# XGBoostJob constants
XGBOOSTJOB_KIND = "XGBoostJob"
XGBOOSTJOB_PLURAL = "xgboostjobs"
XGBOOSTJOB_REPLICA_TYPES = {"master", "worker"}

# MPIJob constants
MPIJOB_KIND = "MPIJob"
MPIJOB_PLURAL = "mpijobs"
MPIJOB_REPLICA_TYPES = {"launcher", "worker"}

# PaddleJob constants
PADDLEJOB_KIND = "PaddleJob"
PADDLEJOB_PLURAL = "paddlejobs"
PADDLEJOB_REPLICA_TYPES = {"master", "worker"}

PADDLEJOB_BASE_IMAGE = (
    "docker.io/paddlepaddle/paddle:2.4.0rc0-gpu-cuda11.2-cudnn8.1-trt8.0"
)

# Dictionary to get plural and model for each Job kind.
JOB_KINDS = {
    TFJOB_KIND: {"plural": TFJOB_PLURAL, "model": models.KubeflowOrgV1TFJob},
    PYTORCHJOB_KIND: {
        "plural": PYTORCHJOB_PLURAL,
        "model": models.KubeflowOrgV1PyTorchJob,
    },
    MXJOB_KIND: {"plural": MXJOB_PLURAL, "model": models.KubeflowOrgV1MXJob},
    XGBOOSTJOB_KIND: {
        "plural": XGBOOSTJOB_PLURAL,
        "model": models.KubeflowOrgV1XGBoostJob,
    },
    MPIJOB_KIND: {"plural": MPIJOB_PLURAL, "model": models.KubeflowOrgV1MPIJob},
    PADDLEJOB_KIND: {
        "plural": PADDLEJOB_PLURAL,
        "model": models.KubeflowOrgV1PaddleJob,
    },
}
