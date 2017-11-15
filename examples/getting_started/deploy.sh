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

# Deploy a cluster on GKE

MODE=deploy
CLUSTER_NAME=dev3
ZONE=us-central1-f
MACHINE_TYPE=n1-standard-8

k8s --mode $MODE \
    --zone $ZONE \
    --machine_type ${MACHINE_TYPE} \
    --cluster_name ${CLUSTER_NAME}

# ISSUE: Currently gives error like the following:
# Error: release tf-job failed: namespaces "default" is forbidden: User "system:serviceaccount:kube-system:tiller" cannot get namespaces in the namespace "default": Unknown user "system:serviceaccount:kube-system:tiller"

# gcloud container clusters create dev1 \
#   --zone=us-central1-f \
#   --cluster-version=1.8.1-gke.1 \
#   --machine-type=n1-standard-8 \
#   --enable-autoscaling --max-nodes=3 --min-nodes=1
#
# gcloud container clusters get-credentials dev1 --zone us-central1-f
#
# kubectl -n kube-system create sa tiller
#
# helm init --service-account tiller
#
# CHART=https://storage.googleapis.com/tf-on-k8s-dogfood-releases/latest/tf-job-operator-chart-latest.tgz
# helm install ${CHART} -n tf-job --wait --replace --set cloud=gke
