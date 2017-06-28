#!/bin/bash

SRC_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

DEPLOY_FILE=${SRC_DIR}/../images/tf_operator/tf_job_operator_deployment.yaml

kubectl delete -f ${DEPLOY_FILE}

kubectl delete thirdpartyresources tf-job.mlkube.io

# delete will throw an error if the resource doesn't exist so we do the deletes before setting exit on error

set -e

${SRC_DIR}/../images/tf_operator/build_and_push.sh

kubectl create -f ${DEPLOY_FILE}