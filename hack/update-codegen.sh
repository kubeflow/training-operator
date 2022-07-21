#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
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

# This shell is used to auto generate some useful tools for k8s, such as lister,
# informer, deepcopy, defaulter and so on.

# Ignore shellcheck for IDEs
# shellcheck disable=SC2116
# shellcheck disable=SC2046
# shellcheck disable=SC2006

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
ROOT_PKG=github.com/kubeflow/training-operator

# Grab code-generator version from go.sum
CODEGEN_VERSION=$(grep 'k8s.io/code-generator' go.mod | awk '{print $2}')
CODEGEN_PKG=$(echo $(go env GOPATH)"/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION}")

if [[ ! -d ${CODEGEN_PKG} ]]; then
    echo "${CODEGEN_PKG} is missing. Running 'go mod download'."
    go mod download
fi

echo ">> Using ${CODEGEN_PKG}"

# Grab openapi-gen version from go.mod
OPENAPI_VERSION=$(grep 'k8s.io/kube-openapi' go.mod | awk '{print $2}')
OPENAPI_PKG=$(echo $(go env GOPATH)"/pkg/mod/k8s.io/kube-openapi@${OPENAPI_VERSION}")

if [[ ! -d ${OPENAPI_PKG} ]]; then
    echo "${OPENAPI_PKG} is missing. Running 'go mod download'."
    go mod download
fi

echo ">> Using ${OPENAPI_PKG}"

# code-generator does work with go.mod but makes assumptions about
# the project living in `$GOPATH/src`. To work around this and support
# any location; create a temporary directory, use this as an output
# base, and copy everything back once generated.
TEMP_DIR=$(mktemp -d)
cleanup() {
    echo ">> Removing ${TEMP_DIR}"
    rm -rf ${TEMP_DIR}
}
trap "cleanup" EXIT SIGINT

echo ">> Temporary output directory ${TEMP_DIR}"

# Ensure we can execute.
chmod +x ${CODEGEN_PKG}/generate-groups.sh

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
cd ${SCRIPT_ROOT}
${CODEGEN_PKG}/generate-groups.sh "all" \
    github.com/kubeflow/training-operator/pkg/client github.com/kubeflow/training-operator/pkg/apis \
    kubeflow.org:v1 \
    --output-base "${TEMP_DIR}" \
    --go-header-file hack/boilerplate/boilerplate.go.txt

# Notice: The code in code-generator does not generate defaulter by default.
# We need to build binary from vendor cmd folder.
#echo "Building defaulter-gen"
#go build -o defaulter-gen ${CODEGEN_PKG}/cmd/defaulter-gen

# ${GOPATH}/bin/defaulter-gen is automatically built from ${CODEGEN_PKG}/generate-groups.sh
echo "Generating defaulters for kubeflow.org/v1"
${GOPATH}/bin/defaulter-gen --input-dirs github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1 \
    -O zz_generated.defaults \
    --output-package github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1 \
    --go-header-file hack/boilerplate/boilerplate.go.txt "$@" \
    --output-base "${TEMP_DIR}"

cd - >/dev/null

# Notice: The code in kube-openapi does not generate defaulter by default.
# We need to build binary from pkg cmd folder.
echo "Building openapi-gen"
go build -o openapi-gen ${OPENAPI_PKG}/cmd/openapi-gen

echo "Generating OpenAPI specification for kubeflow.org/v1"
./openapi-gen --input-dirs github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1,github.com/kubeflow/common/pkg/apis/common/v1 \
    --report-filename=hack/violation_exception.list \
    --output-package github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1 \
    --go-header-file hack/boilerplate/boilerplate.go.txt "$@" \
    --output-base "${TEMP_DIR}"

cd - >/dev/null

# Copy everything back.
cp -a "${TEMP_DIR}/${ROOT_PKG}/." "${SCRIPT_ROOT}/"

# Clean up binaries we build for update codegen
rm ./openapi-gen
