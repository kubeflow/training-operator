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

SWAGGER_JAR_URL="http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.6/swagger-codegen-cli-2.4.6.jar"
SWAGGER_CODEGEN_JAR="hack/python-sdk/swagger-codegen-cli.jar"
SWAGGER_CODEGEN_CONF="hack/python-sdk/swagger_config.json"
SWAGGER_CODEGEN_FILE="pkg/apis/tensorflow/v1/swagger.json"
SDK_OUTPUT_PATH="sdk/python"

if [ -z "${GOPATH:-}" ]; then
    export GOPATH=$(go env GOPATH)
fi

# TBD (@jinchihe) This is not consistent with current generation in hack/update-codegen.sh.
# Need to confirm current one is useful. To aviod breaking current one, backup and rollback.
mv pkg/apis/tensorflow/v1/openapi_generated.go pkg/apis/tensorflow/v1/openapi_generated.go.backup

echo "Generating OpenAPI specification ..."
go run vendor/k8s.io/code-generator/cmd/openapi-gen/main.go --input-dirs github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1,github.com/kubeflow/common/pkg/apis/common/v1 --output-package github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1 --go-header-file hack/boilerplate/boilerplate.go.txt

echo "Generating swagger file ..."
go run hack/python-sdk/main.go 0.1 > ${SWAGGER_CODEGEN_FILE}

echo "Downloading the swagger-codegen JAR package ..."
wget -O ${SWAGGER_CODEGEN_JAR} ${SWAGGER_JAR_URL}

echo "Generating Python SDK for Kubeflow TF-Operator ..."
java -jar ${SWAGGER_CODEGEN_JAR} generate -i ${SWAGGER_CODEGEN_FILE} -l python -o ${SDK_OUTPUT_PATH} -c ${SWAGGER_CODEGEN_CONF}

# Rollback the current openapi_generated.go
mv pkg/apis/tensorflow/v1/openapi_generated.go.backup pkg/apis/tensorflow/v1/openapi_generated.go

echo "Kubeflow TF-Operator Python SDK is generated successfully to folder ${SDK_OUTPUT_PATH}/."
