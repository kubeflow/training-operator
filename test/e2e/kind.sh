#!/bin/bash

# Copyright 2022 IBM, Red Hat
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail
: "${INGRESS_NGINX_VERSION:=controller-v1.9.6}"

BASE_DIR=`pwd`
TEST_DIR="${BASE_DIR}/test/e2e"

echo "Creating KinD cluster"
kind create cluster --name training-operator-cluster --config ${TEST_DIR}/kind-cluster-config.yaml

echo "Deploying Ingress controller into KinD cluster"
curl https://raw.githubusercontent.com/kubernetes/ingress-nginx/"${INGRESS_NGINX_VERSION}"/deploy/static/provider/kind/deploy.yaml | sed "s/--publish-status-address=localhost/--report-node-internal-ip-address\\n        - --status-update-interval=10/g" | kubectl apply -f -
kubectl annotate ingressclass nginx "ingressclass.kubernetes.io/is-default-class=true"
kubectl -n ingress-nginx wait --timeout=300s --for=condition=Available deployments --all
