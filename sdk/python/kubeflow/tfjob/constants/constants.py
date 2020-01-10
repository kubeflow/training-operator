# Copyright 2019 kubeflow.org.
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

# TFJob K8S constants
TFJOB_GROUP = 'kubeflow.org'
TFJOB_KIND = 'TFJob'
TFJOB_PLURAL = 'tfjobs'
TFJOB_VERSION = os.environ.get('TFJOB_VERSION', 'v1')

TFJOB_LOGLEVEL = os.environ.get('TFJOB_LOGLEVEL', 'INFO').upper()

# How long to wait in seconds for requests to the ApiServer
APISERVER_TIMEOUT = 120

#TFJob Labels Name
TFJOB_CONTROLLER_LABEL = 'controller-name'
TFJOB_GROUP_LABEL = 'group-name'
TFJOB_NAME_LABEL = 'tf-job-name'
TFJOB_TYPE_LABEL = 'tf-replica-type'
TFJOB_INDEX_LABEL = 'tf-replica-index'
TFJOB_ROLE_LABEL = 'job-role'
