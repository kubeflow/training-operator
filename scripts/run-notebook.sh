#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
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

# This bash script is used to run the example notebooks

NOTEBOOK_INPUT=""
NOTEBOOK_OUTPUT=""
PAPERMILL_PARAMS=()
PAPERMILL_PARAM_YAML=""
PYTHON_SDK="git+https://github.com/kubeflow/training-operator.git#subdirectory=sdk/python"

usage() {
  echo "Usage: $0 -i <input_notebook> -o <output_notebook> [-p \"<param> <value>\"...] [-y <params.yaml>]"
  echo "Options:"
  echo "  -i  Input notebook (required)"
  echo "  -o  Output notebook (required)"
  echo "  -p  Papermill parameters (optional), pass param name and value pair (in quotes whitespace separated)"
  echo "  -y  Papermill parameters YAML file (optional)"
  echo "  -k  Kubeflow Training Operator Python SDK (optional)"
  echo "  -h  Show this help message"
  exit 1
}

while getopts "i:o:y:p:k:h:" opt; do
  case "$opt" in
    i) NOTEBOOK_INPUT="$OPTARG" ;;          # -i for notebook input path
    o) NOTEBOOK_OUTPUT="$OPTARG" ;;         # -o for notebook output path
    p) PAPERMILL_PARAMS+=("$OPTARG") ;;     # -p for papermill parameters
    y) PAPERMILL_PARAM_YAML="$OPTARG" ;;    # -y for papermill parameter yaml path
    k) PYTHON_SDK="$OPTARG" ;;              # -k for training operator python sdk
    h) usage ;;                             # -h for help (usage)
    *) usage; exit 1 ;;
  esac
done

if [ -z "$NOTEBOOK_INPUT" ] || [ -z "$NOTEBOOK_OUTPUT" ]; then
  echo "Error: -i notebook input path and -o notebook output path are required."
  exit 1
fi

# Install Python dependencies to run Jupyter Notebooks with papermill
pip install jupyter ipykernel papermill==2.6.0

papermill_cmd="papermill $NOTEBOOK_INPUT $NOTEBOOK_OUTPUT -p python_sdk $PYTHON_SDK"

# Add papermill parameters (param name and value)
for param in "${PAPERMILL_PARAMS[@]}"; do
  papermill_cmd="$papermill_cmd -p $param"
done

if [ -n "$PAPERMILL_PARAM_YAML" ]; then
  papermill_cmd="$papermill_cmd -y $PAPERMILL_PARAM_YAML"
fi

echo "Running command: $papermill_cmd"
$papermill_cmd

if [ $? -ne 0 ]; then
  echo "Error: papermill execution failed." >&2
  exit 1
fi

echo "Notebook execution completed successfully. Output saved to $NOTEBOOK_OUTPUT"
