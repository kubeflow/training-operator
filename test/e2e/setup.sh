set -euo pipefail
: "${KUEUE_VERSION:=v0.6.1}" # v0.6.1 is current release
: "${KFTO_IMG:=training-operator:dev}" 

BASE_DIR=`pwd`
TEST_DIR="${BASE_DIR}/test/e2e"

echo "Installing Kueue into the cluster"
kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/${KUEUE_VERSION}/manifests.yaml

echo "Wait for Kueue deployment"
kubectl -n kueue-system wait --timeout=300s --for=condition=Available deployments --all

echo "Creating Kueue Resources"
kubectl apply -f ${TEST_DIR}/kueue-config.yaml

echo "Build training operator image"
docker build -t ${KFTO_IMG} -f ${BASE_DIR}/build/images/training-operator/Dockerfile .

echo "Load training operator image into cluster"
kind load --name training-operator-cluster docker-image training-operator:dev
KFTO_IMG=training-operator:dev make deploy