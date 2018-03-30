#!/bin/bash
#
# A simple script to submit the Argo workflow to build the release.
#
# Usage submit_release_job.sh ${COMMIT}
#
# COMMIT=commit to build at
set -ex

COMMIT=$1

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

JOB_NAME="tf-operator-release"
JOB_TYPE=tf-operator-release
BUILD_NUMBER=$(uuidgen)
BUILD_NUMBER=${BUILD_NUMBER:0:4}
REPO_OWNER=kubeflow
REPO_NAME=tf-operator
ENV=releasing
DATE=`date +%Y%m%d`
PULL_BASE_SHA=${COMMIT:0:8}
VERSION_TAG="v${DATE}-${PULL_BASE_SHA}"


PROW_VAR="JOB_NAME=${JOB_NAME},JOB_TYPE=${JOB_TYPE},REPO_NAME=${REPO_NAME}"
PROW_VAR="${PROW_VAR},REPO_OWNER=${REPO_OWNER},BUILD_NUMBER=${BUILD_NUMBER}" 
PROW_VAR="${PROW_VAR},PULL_BASE_SHA=${PULL_BASE_SHA}" 

cd ${ROOT}/test/workflows

ks param set --env=${ENV} workflows namespace kubeflow-releasing
ks param set --env=${ENV} workflows name "${USER}-${JOB_NAME}-${PULL_BASE_SHA}-${BUILD_NUMBER}"
ks param set --env=${ENV} workflows prow_env "${PROW_VAR}"
ks param set --env=${ENV} workflows versionTag "${VERSION_TAG}"
ks apply ${ENV} -c workflows
