#!/bin/bash

# Copyright 2025 The Kubeflow Authors.
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

# The script is used to build images needed to run JAX Job E2E test.


set -o errexit
set -o nounset
set -o pipefail

# Build Image for MNIST example with SPMD for JAX
docker build examples/jax/jax-dist-spmd-mnist -t ${JAX_JOB_CI_IMAGE} -f examples/jax/jax-dist-spmd-mnist/Dockerfile
