#!/usr/bin/env python

# Copyright 2021 The Kubeflow Authors.
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

"""
This script is used for updating generated SDK files.
"""

import os
import fileinput
import re

__replacements = [
    ("import kubeflow.training", "from kubeflow.training.models import *"),
    ("kubeflow.training.models.v1\/.*.v1.", "V1"),
    ("kubeflow.training.models.kubeflow/org/v1/", "kubeflow_org_v1_"),
    ("\.kubeflow.org.v1\.", ".KubeflowOrgV1"),
]

sdk_dir = os.path.abspath(os.path.join(__file__, "../../..", "sdk/python"))


def main():
    fix_test_files()
    add_imports()


def fix_test_files() -> None:
    """
    Fix invalid model imports in generated model tests
    """
    test_folder_dir = os.path.join(sdk_dir, "test")
    test_files = os.listdir(test_folder_dir)
    for test_file in test_files:
        print(f"Precessing file {test_file}")
        if test_file.endswith(".py"):
            with fileinput.FileInput(
                os.path.join(test_folder_dir, test_file), inplace=True
            ) as file:
                for line in file:
                    print(_apply_regex(line), end="")


def add_imports() -> None:
    with open(os.path.join(sdk_dir, "kubeflow/training/__init__.py"), "a") as f:
        f.write("from kubeflow.training.api.training_client import TrainingClient\n")
    with open(os.path.join(sdk_dir, "kubeflow/__init__.py"), "a") as f:
        f.write("__path__ = __import__('pkgutil').extend_path(__path__, __name__)")

    # Add Kubernetes models to proper deserialization of Training models.
    with open(os.path.join(sdk_dir, "kubeflow/training/models/__init__.py"), "a") as f:
        f.write("\n")
        f.write("# Import Kubernetes models.\n")
        f.write("from kubernetes.client import V1ObjectMeta\n")
        f.write("from kubernetes.client import V1ListMeta\n")
        f.write("from kubernetes.client import V1ManagedFieldsEntry\n")
        f.write("from kubernetes.client import V1PodTemplateSpec\n")
        f.write("from kubernetes.client import V1PodSpec\n")
        f.write("from kubernetes.client import V1Container\n")
        f.write("from kubernetes.client import V1ResourceRequirements\n")


def _apply_regex(input_str: str) -> str:
    for pattern, replacement in __replacements:
        input_str = re.sub(pattern, replacement, input_str)
    return input_str


if __name__ == "__main__":
    main()
