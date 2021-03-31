#!/bin/bash
#
# A simple script to recreate the Kubeflow test app
#
set -ex
# Create a namespace for kubeflow deployment
NAMESPACE=kubeflow

# Which version of Kubeflow to use
# For a list of releases refer to:
# https://github.com/kubeflow/kubeflow/releases
VERSION=master
API_VERSION=v1.7.0

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd ${DIR}

APP_NAME=test-app


if [ -d ${DIR}/${APP_NAME} ]; then
	# TODO(jlewi): Maybe we should prompt to ask if we want to delete?
	echo "Directory ${DIR}/${APP_NAME} exists"
	echo "Do you want to delete ${DIR}/${APP_NAME} y/n[n]:"
	read response

	if [ "${response}"=="y" ]; then
		rm -r ${DIR}/${APP_NAME}
	else
		"Aborting"
		exit 1
	fi
fi

ks init ${APP_NAME} --api-spec=version:${API_VERSION}
cd ${APP_NAME}
ks env set default --namespace ${NAMESPACE}

# Install Kubeflow components
ks registry add kubeflow github.com/kubeflow/kubeflow/tree/${VERSION}/kubeflow

ks pkg install kubeflow/core@${VERSION}

# Create templates for core components
ks generate kubeflow-core core

# Run autoformat from the git root
cd ${DIR}/..
bash <(curl -s https://raw.githubusercontent.com/kubeflow/kubeflow/${VERSION}/scripts/autoformat_jsonnet.sh)
