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
from typing import Final

# General constants
# How long to wait in seconds for requests to the ApiServer
APISERVER_TIMEOUT: Final[int] = 120
KUBEFLOW_GROUP: Final[str] = "kubeflow.org"

# TFJob K8S constants
TFJOB_KIND: Final[str] = "TFJob"
TFJOB_PLURAL: Final[str] = "tfjobs"
TFJOB_VERSION: Final[str] = os.environ.get("TFJOB_VERSION", "v1")

TFJOB_LOGLEVEL: Final[str] = os.environ.get("TFJOB_LOGLEVEL", "INFO").upper()

TFJOB_BASE_IMAGE: Final[str] = "docker.io/tensorflow/tensorflow:2.9.1"
TFJOB_BASE_IMAGE_GPU: Final[str] = "docker.io/tensorflow/tensorflow:2.9.1-gpu"

# Job Label Names
JOB_GROUP_LABEL: Final[str] = "group-name"
JOB_NAME_LABEL: Final[str] = "training.kubeflow.org/job-name"
JOB_TYPE_LABEL: Final[str] = "training.kubeflow.org/replica-type"
JOB_INDEX_LABEL: Final[str] = "training.kubeflow.org/replica-index"
JOB_ROLE_LABEL: Final[str] = "training.kubeflow.org/job-role"
JOB_ROLE_MASTER: Final[str] = "master"

JOB_STATUS_SUCCEEDED: Final[str] = "Succeeded"
JOB_STATUS_FAILED: Final[str] = "Failed"
JOB_STATUS_RUNNING: Final[str] = "Running"

# PyTorchJob K8S constants
PYTORCHJOB_KIND: Final[str] = "PyTorchJob"
PYTORCHJOB_PLURAL: Final[str] = "pytorchjobs"
PYTORCHJOB_VERSION: Final[str] = os.environ.get("PYTORCHJOB_VERSION", "v1")

PYTORCH_LOGLEVEL: Final[str] = os.environ.get("PYTORCHJOB_LOGLEVEL", "INFO").upper()

PYTORCHJOB_BASE_IMAGE: Final[
    str
] = "docker.io/pytorch/pytorch:1.12.1-cuda11.3-cudnn8-runtime"

# XGBoostJob K8S constants
XGBOOSTJOB_KIND: Final[str] = "XGBoostJob"
XGBOOSTJOB_PLURAL: Final[str] = "xgboostjobs"
XGBOOSTJOB_VERSION: Final[str] = os.environ.get("XGBOOSTJOB_VERSION", "v1")

XGBOOST_LOGLEVEL: Final[str] = os.environ.get("XGBOOSTJOB_LOGLEVEL", "INFO").upper()

# MPIJob K8S constants
MPIJOB_KIND: Final[str] = "MPIJob"
MPIJOB_PLURAL: Final[str] = "mpijobs"
MPIJOB_VERSION: Final[str] = os.environ.get("MPIJOB_VERSION", "v1")

MPI_LOGLEVEL: Final[str] = os.environ.get("MPIJOB_LOGLEVEL", "INFO").upper()

# MXNETJob K8S constants
MXJOB_KIND: Final[str] = "MXJob"
MXJOB_PLURAL: Final[str] = "mxjobs"
MXJOB_VERSION: Final[str] = os.environ.get("MXJOB_VERSION", "v1")

MX_LOGLEVEL: Final[str] = os.environ.get("MXJOB_LOGLEVEL", "INFO").upper()
