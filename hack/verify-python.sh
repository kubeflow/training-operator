#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")/..
DIFFROOT="${SCRIPT_ROOT}/sdk"
TMP_DIFFROOT="${SCRIPT_ROOT}/_tmp/sdk"
_tmp="${SCRIPT_ROOT}/_tmp"

cleanup() {
  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_DIFFROOT}"
cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

black "${DIFFROOT}" --exclude '/*kubeflow_org_v1*|__init__.py|api_client.py|configuration.py|exceptions.py|rest.py'

echo "Comparing ${DIFFROOT} with ${TMP_DIFFROOT}..."
ret=0
diff -Naupr "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?

cp -a "${TMP_DIFFROOT}"/* "${DIFFROOT}"

if [[ $ret -eq 0 ]]; then
  echo "Code is properly formatted."
else
  echo "Code is not properly formatted. Please run 'make fmt-python'"
  exit 1
fi

if ! flake8 "${DIFFROOT}" --exclude='/*kubeflow_org_v1*,__init__.py,api_client.py,configuration.py,exceptions.py,rest.py'; then
  echo "Code linting failed. Please fix the issues reported by flake8."
  exit 1
fi
