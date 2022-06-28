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

# General constants
# How long to wait in seconds for requests to the ApiServer
APISERVER_TIMEOUT = 120

# TFJob K8S constants
TFJOB_GROUP = 'kubeflow.org'
TFJOB_KIND = 'TFJob'
TFJOB_PLURAL = 'tfjobs'
TFJOB_VERSION = os.environ.get('TFJOB_VERSION', 'v1')

TFJOB_LOGLEVEL = os.environ.get('TFJOB_LOGLEVEL', 'INFO').upper()

# Job Label Names
JOB_GROUP_LABEL = 'group-name'
JOB_NAME_LABEL = 'training.kubeflow.org/job-name'
JOB_TYPE_LABEL = 'training.kubeflow.org/replica-type'
JOB_INDEX_LABEL = 'training.kubeflow.org/replica-index'
JOB_ROLE_LABEL = 'training.kubeflow.org/job-role'

JOB_STATUS_SUCCEEDED = 'Succeeded'
JOB_STATUS_FAILED = 'Failed'
JOB_STATUS_RUNNING = 'Running'

# PyTorchJob K8S constants
PYTORCHJOB_GROUP = 'kubeflow.org'
PYTORCHJOB_KIND = 'PyTorchJob'
PYTORCHJOB_PLURAL = 'pytorchjobs'
PYTORCHJOB_VERSION = os.environ.get('PYTORCHJOB_VERSION', 'v1')

PYTORCH_LOGLEVEL = os.environ.get('PYTORCHJOB_LOGLEVEL', 'INFO').upper()

# XGBoostJob K8S constants
XGBOOSTJOB_GROUP = 'kubeflow.org'
XGBOOSTJOB_KIND = 'XGBoostJob'
XGBOOSTJOB_PLURAL = 'xgboostjobs'
XGBOOSTJOB_VERSION = os.environ.get('XGBOOSTJOB_VERSION', 'v1')

XGBOOST_LOGLEVEL = os.environ.get('XGBOOSTJOB_LOGLEVEL', 'INFO').upper()

# MPIJob K8S constants
MPIJOB_GROUP = 'kubeflow.org'
MPIJOB_KIND = 'MPIJob'
MPIJOB_PLURAL = 'mpijobs'
MPIJOB_VERSION = os.environ.get('MPIJOB_VERSION', 'v1')

MPI_LOGLEVEL = os.environ.get('MPIJOB_LOGLEVEL', 'INFO').upper()

# MXNETJob K8S constants
MXJOB_GROUP = 'kubeflow.org'
MXJOB_KIND = 'MXJob'
MXJOB_PLURAL = 'mxjobs'
MXJOB_VERSION = os.environ.get('MXJOB_VERSION', 'v1')

MX_LOGLEVEL = os.environ.get('MXJOB_LOGLEVEL', 'INFO').upper()
