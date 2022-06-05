# Copyright 2022 The Kubeflow Authors.
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

go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

setup-envtest use -p path ${KUBERNETES_VERSION}
eval $(setup-envtest use -i -p env ${KUBERNETES_VERSION})
go test ./... -coverprofile cover.out 
