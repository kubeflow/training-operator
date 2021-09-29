#!/usr/bin/env python

"""
This script is used for updating generated SDK files.
"""

import os
import fileinput
import re

__replacements = [
    ("import kubeflow.training", "from kubeflow.training.models import *"),
    ("kubeflow.training.models.v1\/.*.v1.", "V1")
]

sdk_dir = os.path.abspath(os.path.join(__file__, "../../..", "sdk/python"))


def main():
    fix_test_files()


def fix_test_files() -> None:
    """
    Fix invalid model imports in generated model tests
    """
    os.path.realpath(__file__)
    test_folder_dir = os.path.join(sdk_dir, "test")
    test_files = os.listdir(test_folder_dir)
    for test_file in test_files:
        print(f"Precessing file {test_file}")
        if test_file.endswith(".py"):
            with fileinput.FileInput(os.path.join(test_folder_dir, test_file), inplace=True) as file:
                for line in file:
                    print(_apply_regex(line), end='')


def _apply_regex(input_str: str) -> str:
    for pattern, replacement in __replacements:
        input_str = re.sub(pattern, replacement, input_str)
    return input_str


if __name__ == '__main__':
    main()
