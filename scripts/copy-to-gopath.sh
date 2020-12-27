#!/bin/bash

# Copyright 2018 The Kubeflow Authors.
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

# This shell script is used to build an image from our argo workflow

set -o errexit
set -o nounset
set -o pipefail

export PATH=${GOPATH}/bin:/usr/local/go/bin:${PATH}
GO_DIR=${GOPATH}/src/github.com/kubeflow/${REPO_NAME}

# e2e test will run in go_dir. this is a required step.
echo "Create symlink to GOPATH"
# TODO(@Jeffwan): it should be ${REPO_OWNER}. Change it back later.
mkdir -p ${GOPATH}/src/github.com/kubeflow
ln -s ${PWD} ${GO_DIR}
cd ${GO_DIR}
