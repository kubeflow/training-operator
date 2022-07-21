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

TIMEOUT=30
until kubectl get pods -n kubeflow | grep training-operator | grep 1/1 || [[ $TIMEOUT -eq 1 ]]; do
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done
kubectl version
kubectl cluster-info
kubectl get nodes
kubectl get pods -n kubeflow
kubectl describe pods -n kubeflow
