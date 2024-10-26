#!/usr/bin/env bash

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

# Run this script from the root location: `make generate`

set -o errexit
set -o nounset

SWAGGER_JAR_URL="https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/4.3.1/openapi-generator-cli-4.3.1.jar"
SWAGGER_CODEGEN_JAR="hack/python-sdk-v2/openapi-generator-cli.jar"
SWAGGER_CODEGEN_CONF="hack/python-sdk-v2/swagger_config.json"
SWAGGER_CODEGEN_FILE="api.v2/openapi-spec/swagger.json"

SDK_OUTPUT_PATH="sdk_v2"

if [[ ! -f "$SWAGGER_CODEGEN_JAR" ]]; then
  echo "Downloading the swagger-codegen JAR package ..."
  wget -O "${SWAGGER_CODEGEN_JAR}" ${SWAGGER_JAR_URL}
fi

echo "Generating Python SDK for Training Operator V2 ..."
java -jar "${SWAGGER_CODEGEN_JAR}" generate -i "${SWAGGER_CODEGEN_FILE}" -g python -o "${SDK_OUTPUT_PATH}" -c "${SWAGGER_CODEGEN_CONF}"

echo "Removing unused files for the Python SDK"

git clean -f ${SDK_OUTPUT_PATH}/.openapi-generator
git clean -f ${SDK_OUTPUT_PATH}/.gitignore
git clean -f ${SDK_OUTPUT_PATH}/.gitlab-ci.yml
git clean -f ${SDK_OUTPUT_PATH}/git_push.sh
git clean -f ${SDK_OUTPUT_PATH}/.openapi-generator-ignore
git clean -f ${SDK_OUTPUT_PATH}/.travis.yml
git clean -f ${SDK_OUTPUT_PATH}/README.md
git clean -f ${SDK_OUTPUT_PATH}/requirements.txt
git clean -f ${SDK_OUTPUT_PATH}/setup.cfg
git clean -f ${SDK_OUTPUT_PATH}/setup.py
git clean -f ${SDK_OUTPUT_PATH}/test-requirements.txt
git clean -f ${SDK_OUTPUT_PATH}/tox.ini

# TODO (andreyvelich): Discuss if we should use these test files.
git clean -f ${SDK_OUTPUT_PATH}/test
