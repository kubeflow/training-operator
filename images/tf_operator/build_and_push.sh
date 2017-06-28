#!/bin/bash
set -e

SRC_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR=${SRC_DIR}/../../

. ${ROOT_DIR}/config.sh

IMAGE=${REGISTRY}/tf_operator:latest

DIR=`mktemp -d`
echo Use ${DIR} as context
go install mlkube.io/cmd/tf_operator
cp ${GOPATH}/bin/tf_operator ${DIR}/
cp ${SRC_DIR}/Dockerfile ${DIR}/

docker build -t $IMAGE -f ${DIR}/Dockerfile ${DIR}
gcloud docker -- push $IMAGE
echo pushed $IMAGE
