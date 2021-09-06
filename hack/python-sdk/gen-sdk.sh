#!/usr/bin/env bash

# Copyright 2019 The Kubeflow Authors.
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

SWAGGER_JAR_URL="https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/4.3.1/openapi-generator-cli-4.3.1.jar"
SWAGGER_CODEGEN_JAR="hack/python-sdk/openapi-generator-cli.jar"
SWAGGER_CODEGEN_CONF="hack/python-sdk/swagger_config.json"
SDK_OUTPUT_PATH="/tmp/sdk/python"
FRAMEWORKS=(tensorflow pytorch mxnet xgboost)
VERSION=1.3.0

if [ -z "${GOPATH:-}" ]; then
    export GOPATH=$(go env GOPATH)
fi

echo "Generating OpenAPI specification ..."
echo "./hack/update-codegen.sh already help us generate openapi specs ..."

echo "Downloading the swagger-codegen JAR package ..."
wget -O ${SWAGGER_CODEGEN_JAR} ${SWAGGER_JAR_URL}


for FRAMEWORK in ${FRAMEWORKS[@]}; do
    SWAGGER_CODEGEN_FILE="pkg/apis/${FRAMEWORK}/v1/swagger.json"

    echo "Generating swagger file for ${FRAMEWORK} ..."
    go run hack/python-sdk/main.go ${FRAMEWORK} ${VERSION} > ${SWAGGER_CODEGEN_FILE}
    
    echo "Generating Python SDK for ${FRAMEWORK} ..."
    java -jar ${SWAGGER_CODEGEN_JAR} generate -i ${SWAGGER_CODEGEN_FILE} -g python -o ${SDK_OUTPUT_PATH} -c ${SWAGGER_CODEGEN_CONF}
done

echo "Kubeflow Training Operator Python SDK is generated successfully to folder ${SDK_OUTPUT_PATH}/."