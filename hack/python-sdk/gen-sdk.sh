#!/usr/bin/env bash

# Copyright 2021 The Kubeflow Authors.
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

set -o errexit
set -o nounset
set -o pipefail

repo_root="$(realpath "$(dirname "$0")/../..")"

SWAGGER_JAR_URL="https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/4.3.1/openapi-generator-cli-4.3.1.jar"
SWAGGER_CODEGEN_JAR="${repo_root}/hack/python-sdk/openapi-generator-cli.jar"
SWAGGER_CODEGEN_CONF="${repo_root}/hack/python-sdk/swagger_config.json"
SDK_OUTPUT_PATH="${repo_root}/sdk/python"
FRAMEWORKS=(tensorflow pytorch mxnet xgboost)
VERSION=1.3.0

if [ -z "${GOPATH:-}" ]; then
    export GOPATH=$(go env GOPATH)
fi

echo "Generating OpenAPI specification ..."
echo "./hack/update-codegen.sh already help us generate openapi specs ..."

if [[ ! -f "$SWAGGER_CODEGEN_JAR" ]]; then
  echo "Downloading the swagger-codegen JAR package ..."
  wget -O "${SWAGGER_CODEGEN_JAR}" ${SWAGGER_JAR_URL}
fi


for FRAMEWORK in ${FRAMEWORKS[@]}; do
    SWAGGER_CODEGEN_FILE="pkg/apis/${FRAMEWORK}/v1/swagger.json"
    echo "Generating swagger file for ${FRAMEWORK} ..."
    go run "${repo_root}"/hack/python-sdk/main.go "${FRAMEWORK}" ${VERSION} > "${SWAGGER_CODEGEN_FILE}"
done

echo "Merging swagger files from different frameworks into one"
download_url=$(curl -s https://api.github.com/repos/go-swagger/go-swagger/releases/latest | \
  jq -r '.assets[] | select(.name | contains("'"$(uname | tr '[:upper:]' '[:lower:]')"'_amd64")) | .browser_download_url')
curl -o /tmp/swagger -L'#' "$download_url"
chmod +x /tmp/swagger

# it will report warning like 'v1.SchedulingPolicy' already exists in primary or higher priority mixin, skipping
# error code is not 0 but t's acceptable.
/tmp/swagger mixin "${repo_root}"/pkg/apis/tensorflow/v1/swagger.json "${repo_root}"/pkg/apis/pytorch/v1/swagger.json \
  "${repo_root}"/pkg/apis/mxnet/v1/swagger.json "${repo_root}"/pkg/apis/xgboost/v1/swagger.json \
  --output "${repo_root}"/hack/python-sdk/swagger.json --quiet || true

echo "Removing previously generated files ..."
rm -rf "${SDK_OUTPUT_PATH}"/docs/V1*.md "${SDK_OUTPUT_PATH}"/kubeflow/training/models "${SDK_OUTPUT_PATH}"/kubeflow/training/*.py "${SDK_OUTPUT_PATH}"/test/*.py
echo "Generating Python SDK for Training Operator ..."
java -jar "${SWAGGER_CODEGEN_JAR}" generate -i "${repo_root}"/hack/python-sdk/swagger.json -g python -o "${SDK_OUTPUT_PATH}" -c "${SWAGGER_CODEGEN_CONF}"

echo "Kubeflow Training Operator Python SDK is generated successfully to folder ${SDK_OUTPUT_PATH}/."
rm /tmp/swagger

echo "Running post-generation script ..."
"${repo_root}"/hack/python-sdk/post_gen.py
