# Copyright 2021 kubeflow.org.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import textwrap
import inspect
from typing import Callable, List, Dict, Any
from kubernetes import client

from kubeflow.training.constants import constants


def is_running_in_k8s():
    return os.path.isdir("/var/run/secrets/kubernetes.io/")


def get_current_k8s_namespace():
    with open("/var/run/secrets/kubernetes.io/serviceaccount/namespace", "r") as f:
        return f.readline()


def get_default_target_namespace():
    if not is_running_in_k8s():
        return "default"
    return get_current_k8s_namespace()


def set_tfjob_namespace(tfjob):
    tfjob_namespace = tfjob.metadata.namespace
    namespace = tfjob_namespace or get_default_target_namespace()
    return namespace


def set_pytorchjob_namespace(pytorchjob):
    pytorchjob_namespace = pytorchjob.metadata.namespace
    namespace = pytorchjob_namespace or get_default_target_namespace()
    return namespace


def set_xgboostjob_namespace(xgboostjob):
    xgboostjob_namespace = xgboostjob.metadata.namespace
    namespace = xgboostjob_namespace or get_default_target_namespace()
    return namespace


def set_mpijob_namespace(mpijob):
    mpijob_namespace = mpijob.metadata.namespace
    namespace = mpijob_namespace or get_default_target_namespace()
    return namespace


def set_mxjob_namespace(mxjob):
    mxjob_namespace = mxjob.metadata.namespace
    namespace = mxjob_namespace or get_default_target_namespace()
    return namespace


def get_job_labels(name, master=False, replica_type=None, replica_index=None):
    """
    Get labels according to specified flags.
    :param name: job name
    :param master: if need include label 'training.kubeflow.org/job-role: master'.
    :param replica_type: Replica type according to the job type (master, worker, chief, ps etc).
    :param replica_index: Can specify replica index to get one pod of the job.
    :return: Dict: Labels
    """
    labels = {
        constants.JOB_GROUP_LABEL: "kubeflow.org",
        constants.JOB_NAME_LABEL: name,
    }
    if master:
        labels[constants.JOB_ROLE_LABEL] = "master"

    if replica_type:
        labels[constants.JOB_TYPE_LABEL] = str.lower(replica_type)

    if replica_index:
        labels[constants.JOB_INDEX_LABEL] = replica_index

    return labels


def to_selector(labels):
    """
    Transfer Labels to selector.
    """
    parts = []
    for key in labels.keys():
        parts.append("{0}={1}".format(key, labels[key]))

    return ",".join(parts)


def get_script_for_python_packages(packages_to_install, pip_index_url):
    packages_str = " ".join([str(package) for package in packages_to_install])

    script_for_python_packages = textwrap.dedent(
        f"""
        if ! [ -x "$(command -v pip)" ]; then
            python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip
        fi

        PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet \
        --no-warn-script-location --index-url {pip_index_url} {packages_str}
        """
    )

    return script_for_python_packages


def get_pod_template_spec(
    func: Callable,
    parameters: Dict[str, Any],
    base_image: str,
    container_name: str,
    packages_to_install: List[str],
    pip_index_url: str,
):
    """
    Get Pod template spec from the given function and input parameters.
    """

    # Check if function is callable.
    if not callable(func):
        raise ValueError(
            f"Training function must be callable, got function type: {type(func)}"
        )

    # Extract function implementation.
    func_code = inspect.getsource(func)

    # Function might be defined in some indented scope (e.g. in another function).
    # We need to dedent the function code.
    func_code = textwrap.dedent(func_code)

    # Wrap function code to execute it from the file. For example:
    # def train(parameters):
    #     print('Start Training...')
    # train({'lr': 0.01})
    if parameters is None:
        func_code = f"{func_code}\n{func.__name__}()\n"
    else:
        func_code = f"{func_code}\n{func.__name__}({parameters})\n"

    # Prepare execute script template.
    exec_script = textwrap.dedent(
        """
            program_path=$(mktemp -d)
            read -r -d '' SCRIPT << EOM\n
            {func_code}
            EOM
            printf "%s" "$SCRIPT" > $program_path/ephemeral_script.py
            python3 -u $program_path/ephemeral_script.py"""
    )

    # Add function code to the execute script.
    exec_script = exec_script.format(func_code=func_code)

    # Install Python packages if that is required.
    if packages_to_install is not None:
        exec_script = (
            get_script_for_python_packages(packages_to_install, pip_index_url)
            + exec_script
        )

    # Create Pod template spec.
    pod_template_spec = client.V1PodTemplateSpec(
        metadata=client.V1ObjectMeta(annotations={"sidecar.istio.io/inject": "false"}),
        spec=client.V1PodSpec(
            containers=[
                client.V1Container(
                    name=container_name,
                    image=base_image,
                    command=["bash", "-c"],
                    args=[exec_script],
                )
            ]
        ),
    )

    return pod_template_spec


class TableLogger:
    def __init__(self, header, column_format):
        self.header = header
        self.column_format = column_format
        self.first_call = True

    def __call__(self, *values):
        if self.first_call:
            print(self.header)
            self.first_call = False
        print(self.column_format.format(*values))
