#!/bin/bash

# Copyright 2024 The Kubeflow Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# The script is used to build Kubeflow Training image.


set -o errexit
set -o nounset
set -o pipefail

docker build sdk/python/kubeflow/storage_initializer -t ${STORAGE_INITIALIZER_CI_IMAGE} -f sdk/python/kubeflow/storage_initializer/Dockerfile
