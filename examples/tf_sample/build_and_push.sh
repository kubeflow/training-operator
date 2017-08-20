#!/bin/bash
#
# Script to build the tf_sample and push it to GCS.

set -e

SRC_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR=${SRC_DIR}/../../

. ${ROOT_DIR}/config.sh

# The image tag is based on the githash.
GITHASH=$(git rev-parse --short HEAD)
CHANGES=$(git diff-index --quiet HEAD -- || echo "untracked")
if [ -n "$CHANGES" ]; then
  # Get the hash of the diff.
  DIFFHASH=$(git diff  | sha256sum)
  DIFFHASH=${DIFFHASH:0:7}
  GITHASH=${GITHASH}-dirty-${DIFFHASH}
fi

IMAGE=${REGISTRY}/tf_sample:${GITHASH}

SRC_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

docker build -t $IMAGE -f ${SRC_DIR}/Dockerfile ${SRC_DIR}
gcloud docker -- push $IMAGE
echo pushed $IMAGE

IMAGE=${REGISTRY}/tf_sample_gpu:${GITHASH}
SRC_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

docker build -t $IMAGE -f ${SRC_DIR}/Dockerfile.gpu ${SRC_DIR}
gcloud docker -- push $IMAGE
echo pushed $IMAGE
