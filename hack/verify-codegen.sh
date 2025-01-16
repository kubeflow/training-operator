#!/bin/bash

# Copyright 2024 The Kubeflow Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

CURRENT_DIR=$(dirname "${BASH_SOURCE[0]}")
DIFFROOT="${CURRENT_DIR}"
TMP_DIFFROOT="$(mktemp -d -t "$(basename "$0").XXXXXX")"

cleanup() {
  rm -rf "${TMP_DIFFROOT}"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_DIFFROOT}"
cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

echo $TMP_DIFFROOT
echo $DIFFROOT

# Generate files.
make generate

echo "diffing ${DIFFROOT} against freshly generated codegen"
ret=0
diff -Naupr -x.gitignore "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?

if [[ $ret -eq 0 ]]; then
  echo "${DIFFROOT} up to date."
else
  echo "${DIFFROOT} is out of date. Please run make generate"
  exit 1
fi
