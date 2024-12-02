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
TRAINING_PYTHON_SDK="git+https://github.com/kubeflow/training-operator.git#subdirectory=sdk/python"
PYTHON_DEPENDENCIES=()
PYTHON_REQUIREMENTS_FILE=""

usage() {
  echo "Usage: $0 -i <input_notebook> -o <output_notebook> [-p \"<param> <value>\"...] [-y <params.yaml>] [-d <package>==<version>...] [-r <requirements.txt>]"
  echo "Options:"
  echo "  -i  Input notebook (required)"
  echo "  -o  Output notebook (required)"
  echo "  -p  Papermill parameters (optional), pass param name and value pair (in quotes whitespace separated)"
  echo "  -y  Papermill parameters YAML file (optional)"
  echo "  -k  Kubeflow Training Operator Python SDK (optional)"
  echo "  -d  Python dependencies args (optional)"
  echo "  -r  Python dependencies requirements file (optional)"
  echo "  -h  Show this help message"
  echo "NOTE: papermill, jupyter and ipykernel are required Python dependencies to run Notebooks. Dependencies can be passed through -d or -r"
  exit 1
}

while getopts "i:o:y:p:k:r:d:h:" opt; do
  case "$opt" in
    i) NOTEBOOK_INPUT="$OPTARG" ;;            # -i for notebook input path
    o) NOTEBOOK_OUTPUT="$OPTARG" ;;           # -o for notebook output path
    p) PAPERMILL_PARAMS+=("$OPTARG") ;;       # -p for papermill parameters
    y) PAPERMILL_PARAM_YAML="$OPTARG" ;;      # -y for papermill parameter yaml path
    k) TRAINING_PYTHON_SDK="$OPTARG" ;;       # -k for training operator python sdk
    d) PYTHON_DEPENDENCIES+=("$OPTARG") ;;    # -d for passing python dependencies as args
    r) PYTHON_REQUIREMENTS_FILE="$OPTARG" ;;  # -r for passing python dependencies as requirements file
    h) usage ;;                               # -h for help (usage)
    *) usage; exit 1 ;;
  esac
done

if [ -z "$NOTEBOOK_INPUT" ] || [ -z "$NOTEBOOK_OUTPUT" ]; then
  echo "Error: -i notebook input path and -o notebook output path are required."
  exit 1
fi

# Check if we need to install dependencies
if [ ${#PYTHON_DEPENDENCIES[@]} -gt 0 ] || [ -n "$PYTHON_REQUIREMENTS_FILE" ]; then
  pip_install_cmd="pip install"

  for dep in "${PYTHON_DEPENDENCIES[@]}"; do
    pip_install_cmd="$pip_install_cmd $dep"
  done

  if [ -n "$PYTHON_REQUIREMENTS_FILE" ]; then
    pip_install_cmd="$pip_install_cmd -r $PYTHON_REQUIREMENTS_FILE"
  fi

  echo "Installing Dependencies: $pip_install_cmd"
  eval "$pip_install_cmd"
  if [ $? -ne 0 ]; then
    echo "Error: Failed to install dependencies." >&2
    exit 1
  fi
fi

papermill_cmd="papermill $NOTEBOOK_INPUT $NOTEBOOK_OUTPUT -p training_python_sdk $TRAINING_PYTHON_SDK"
# Add papermill parameters (param name and value)
for param in "${PAPERMILL_PARAMS[@]}"; do
  papermill_cmd="$papermill_cmd -p $param"
done

if [ -n "$PAPERMILL_PARAM_YAML" ]; then
  papermill_cmd="$papermill_cmd -y $PAPERMILL_PARAM_YAML"
fi

# Check if papermill is installed
if ! command -v papermill &> /dev/null; then
  echo "Error: papermill is not installed. Please install papermill to proceed. Python dependencies can be passed through -d or -r args"
  exit 1
fi

echo "Running command: $papermill_cmd"
$papermill_cmd

if [ $? -ne 0 ]; then
  echo "Error: papermill execution failed." >&2
  exit 1
fi

echo "Notebook execution completed successfully. Output saved to $NOTEBOOK_OUTPUT"
