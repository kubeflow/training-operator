#!/usr/bin/env bash
# Copyright 2016 The TensorFlow Authors. All Rights Reserved.
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

die() {
  echo $@
  exit 1
}

# Configurations
DOCKER_IMG_NAME="kubeflow/tf-dist-mnist-test"
DOCKER_IMG_TAG="1.0"

# Use TensorFlow v1.5.0 for Python 2.7 and CPU only as we set num_gpus to 0 in the below
DEFAULT_WHL_FILE_LOCATION="https://storage.googleapis.com/tensorflow/linux/cpu/tensorflow-1.5.0-cp27-none-linux_x86_64.whl"

WHL_FILE_LOCATION=${1}
if [[ -z "${WHL_FILE_LOCATION}" ]]; then
  WHL_FILE_LOCATION=${DEFAULT_WHL_FILE_LOCATION}
  echo "use default whl file location"
fi

# Current script directory
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Build docker image for local distributed TensorFlow cluster.
NO_CACHE_FLAG=""
if [[ ! -z "${TF_DIST_DOCKER_NO_CACHE}" ]] &&
   [[ "${TF_DIST_DOCKER_NO_CACHE}" != "0" ]]; then
  NO_CACHE_FLAG="--no-cache"
fi

# Create docker build context directory.
BUILD_DIR=$(mktemp -d)
echo ""
echo "Using whl file location: ${WHL_FILE_LOCATION}"

cp -r ${DIR}/* "${BUILD_DIR}"/ || \
  die "Failed to copy files to ${BUILD_DIR}"

if [[ $WHL_FILE_LOCATION =~ 'http://' || $WHL_FILE_LOCATION =~ 'https://' ]]; then
    # Download whl file into the build context directory.
    wget -P "${BUILD_DIR}" "${WHL_FILE_LOCATION}" || \
        die "Failed to download tensorflow whl file from URL: ${WHL_FILE_LOCATION}"
else
    cp "${WHL_FILE_LOCATION}" "${BUILD_DIR}"
fi

echo "Building in temporary directory: ${BUILD_DIR}"
# Build docker image for test.
docker build ${NO_CACHE_FLAG} -t ${DOCKER_IMG_NAME}:${DOCKER_IMG_TAG} \
   -f "${BUILD_DIR}/Dockerfile" "${BUILD_DIR}" || \
   die "Failed to build docker image: ${DOCKER_IMG_NAME}"

# Clean up docker build context directory.
rm -rf "${BUILD_DIR}"
