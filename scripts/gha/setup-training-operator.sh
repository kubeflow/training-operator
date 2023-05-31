#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
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

# This shell script is used to build a cluster and create a namespace from our
# argo workflow


set -o errexit
set -o nounset
set -o pipefail

echo "Kind load newly locally built image"
# use cluster name which is used in github actions kind create
kind load docker-image ${TRAINING_CI_IMAGE} --name ${KIND_CLUSTER}

echo "Update training operator manifest with newly built image"
cd manifests/overlays/standalone
kustomize edit set image kubeflow/training-operator=${TRAINING_CI_IMAGE}

echo "Installing training operator manifests"
kustomize build . | kubectl apply -f -

if [ "${GANG_SCHEDULER_NAME}" = "scheduler-plugins" ]; then
  echo "Installing Scheduler Plugins..."
  # We need to use latest helm chart since older helm chart has bugs in RBAC.
  git clone https://github.com/kubernetes-sigs/scheduler-plugins.git
  pushd scheduler-plugins/manifests/install/charts

  # Since https://github.com/kubernetes-sigs/scheduler-plugins/pull/526, the scheduler-plugins switch the API group to 'x-k8s.io'.
  # So we must use the specific commit version to available the older API group, 'sigs.k8s.io'.
  # Details: https://github.com/kubeflow/training-operator/issues/1769
  # TODO: Once we support new API group, we should switch the scheduler-plugins version.
  git checkout df16b76a226e58b6961b30ba800e5a713d433c44

  helm install scheduler-plugins as-a-second-scheduler/
  popd
  rm -rf scheduler-plugins

  echo "Configure gang-scheduling using scheduler-plugins to training-operator"
  kubectl patch -n kubeflow deployments training-operator --type='json' \
    -p='[{"op": "add", "path": "/spec/template/spec/containers/0/command/1", "value": "--gang-scheduler-name=scheduler-plugins"}]'
fi

TIMEOUT=30
until kubectl get pods -n kubeflow | grep training-operator | grep 1/1 || [[ $TIMEOUT -eq 1 ]]; do
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done
if [ "${GANG_SCHEDULER_NAME}" = "scheduler-plugins" ]; then
  kubectl wait pods --for=condition=ready -n scheduler-plugins --timeout "${TIMEOUT}s" --all
fi

kubectl version
kubectl cluster-info
kubectl get nodes
kubectl get pods -n kubeflow
kubectl describe pods -n kubeflow
