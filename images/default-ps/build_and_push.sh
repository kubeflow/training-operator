#!/bin/bash
set -e

SRC_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR=${SRC_DIR}/../../

#. ${ROOT_DIR}/config.sh
REGISTRY="wbuchwalter"
TAGS=("1.1.0" "1.2.0" "1.2.1" "1.3.0" "latest")

for i in "${TAGS[@]}"
do  
    BASEIMAGE="tensorflow/tensorflow:$i"
    IMAGE="${REGISTRY}/mlkube-tensorflow-ps:$i"

    DIR=`mktemp -d`
    echo Use ${DIR} as context
    cp ${SRC_DIR}/Dockerfile ${DIR}/Dockerfile
    cp ${SRC_DIR}/main.py ${DIR}/main.py
    docker build -t $IMAGE --build-arg BASE_IMAGE=$BASEIMAGE -f ${DIR}/Dockerfile ${DIR}
    docker push $IMAGE
done