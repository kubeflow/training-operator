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
import logging
import textwrap
import inspect
from typing import Callable, List, Dict, Any
import json
import threading
import queue
import multiprocessing

from kubernetes import client

from kubeflow.training.constants import constants
from kubeflow.training.api_client import ApiClient


logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)


class StatusLogger:
    """Logger to print Training Job statuses."""

    def __init__(self, header, column_format):
        self.header = header
        self.column_format = column_format
        self.first_call = True

    def __call__(self, *values):
        if self.first_call:
            logging.info(self.header)
            self.first_call = False
        logging.info(self.column_format.format(*values))


class FakeResponse:
    """Fake object of RESTResponse to deserialize
    Ref) https://github.com/kubeflow/katib/pull/1630#discussion_r697877815
    Ref) https://github.com/kubernetes-client/python/issues/977#issuecomment-592030030
    """

    def __init__(self, obj):
        self.data = json.dumps(obj)


def is_running_in_k8s():
    return os.path.isdir("/var/run/secrets/kubernetes.io/")


def get_default_target_namespace():
    if not is_running_in_k8s():
        return "default"
    with open("/var/run/secrets/kubernetes.io/serviceaccount/namespace", "r") as f:
        return f.readline()


def create_job(
    custom_api: client.CustomObjectsApi,
    job: object,
    namespace: str,
    job_kind: str,
    job_plural: str,
):
    """Create the Training Job."""

    try:
        custom_api.create_namespaced_custom_object(
            constants.KUBEFLOW_GROUP,
            constants.OPERATOR_VERSION,
            namespace,
            job_plural,
            job,
        )
    except multiprocessing.TimeoutError:
        raise TimeoutError(
            f"Timeout to create {job_kind}: {namespace}/{job.metadata.name}"
        )
    except Exception:
        raise RuntimeError(
            f"Failed to create {job_kind}: {namespace}/{job.metadata.name}"
        )

    logging.info(f"{job_kind} {namespace}/{job.metadata.name} has been created")


def get_job(
    custom_api: client.CustomObjectsApi,
    api_client: ApiClient,
    name: str,
    namespace: str,
    job_model: object,
    job_kind: str,
    job_plural: str,
    timeout: int,
):
    """Get the Training Job."""

    try:
        thread = custom_api.get_namespaced_custom_object(
            constants.KUBEFLOW_GROUP,
            constants.OPERATOR_VERSION,
            namespace,
            job_plural,
            name,
            async_req=True,
        )
        response = FakeResponse(thread.get(timeout))
        job = api_client.deserialize(response, job_model)
        return job

    except multiprocessing.TimeoutError:
        raise TimeoutError(f"Timeout to get {job_kind}: {namespace}/{name}")
    except Exception:
        raise RuntimeError(f"Failed to get {job_kind}: {namespace}/{name}")


def list_jobs(
    custom_api: client.CustomObjectsApi,
    api_client: ApiClient,
    namespace: str,
    job_model: object,
    job_kind: str,
    job_plural: str,
    timeout: int,
):
    """List the Training Jobs."""

    result = []
    try:
        thread = custom_api.list_namespaced_custom_object(
            constants.KUBEFLOW_GROUP,
            constants.OPERATOR_VERSION,
            namespace,
            job_plural,
            async_req=True,
        )
        response = thread.get(timeout)
        result = [
            api_client.deserialize(FakeResponse(item), job_model)
            for item in response.get("items")
        ]
    except multiprocessing.TimeoutError:
        raise TimeoutError(f"Timeout to list {job_kind}s in namespace: {namespace}")
    except Exception:
        raise RuntimeError(f"Failed to list {job_kind}s in namespace: {namespace}")
    return result


def delete_job(
    custom_api: client.CustomObjectsApi,
    name: str,
    namespace: str,
    job_kind: str,
    job_plural: str,
    delete_options: client.V1DeleteOptions,
):
    """Delete the Training Job."""

    try:
        custom_api.delete_namespaced_custom_object(
            constants.KUBEFLOW_GROUP,
            constants.OPERATOR_VERSION,
            namespace,
            job_plural,
            name=name,
            body=delete_options,
        )
    except multiprocessing.TimeoutError:
        raise TimeoutError(f"Timeout to delete {job_kind}: {namespace}/{name}")
    except Exception:
        raise RuntimeError(f"Failed to delete {job_kind}: {namespace}/{name}")

    logging.info(f"{job_kind} {namespace}/{name} has been deleted")


def patch_job(
    custom_api: client.CustomObjectsApi,
    job: object,
    name: str,
    namespace: str,
    job_kind: str,
    job_plural: str,
):
    """Patch the Training Job."""

    try:
        custom_api.patch_namespaced_custom_object(
            constants.KUBEFLOW_GROUP,
            constants.OPERATOR_VERSION,
            namespace,
            job_plural,
            name,
            job,
        )
    except multiprocessing.TimeoutError:
        raise TimeoutError(
            f"Timeout to patch {job_kind}: {namespace}/{job.metadata.name}"
        )
    except Exception:
        raise RuntimeError(
            f"Failed to patch {job_kind}: {namespace}/{job.metadata.name}"
        )

    logging.info(f"{job_kind} {namespace}/{job.metadata.name} has been patched")


def wrap_log_stream(q, stream):
    while True:
        try:
            logline = next(stream)
            q.put(logline)
        except StopIteration:
            q.put(None)
            return


def get_log_queue_pool(streams):
    pool = []
    for stream in streams:
        q = queue.Queue(maxsize=100)
        pool.append(q)
        threading.Thread(target=wrap_log_stream, args=(q, stream)).start()
    return pool


def has_condition(conditions: object, condition_type: str):
    """Verify if the condition list has the required condition.
    Condition should be valid object with `type` and `status`.
    """

    for c in conditions:
        if c.type == condition_type and c.status == constants.CONDITION_STATUS_TRUE:
            return True
    return False


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
