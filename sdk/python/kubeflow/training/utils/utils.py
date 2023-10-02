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
from typing import Optional, Callable, List, Dict, Any
import json
import threading
import queue

from kubeflow.training.constants import constants
from kubeflow.training import models


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


def has_condition(conditions: List[models.V1JobCondition], condition_type: str) -> bool:
    """
    Verify if the condition list has the required condition.
    Condition should be valid object with `type` and `status`.
    """

    for c in conditions:
        if c.type == condition_type and c.status == constants.CONDITION_STATUS_TRUE:
            return True
    return False


def get_script_for_python_packages(
    packages_to_install: List[str], pip_index_url: str
) -> str:
    """
    Get init script to install Python packages from the given pip index URL.
    """
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
    job_kind: str,
    base_image: Optional[str] = None,
    train_func: Optional[Callable] = None,
    parameters: Optional[Dict[str, Any]] = None,
    packages_to_install: Optional[List[str]] = None,
    pip_index_url: str = constants.DEFAULT_PIP_INDEX_URL,
):
    """
    Get Pod template spec for the given function and base image.
    """

    # Assign the default base image.
    # TODO (andreyvelich): Add base image for other Job kinds.
    if base_image is None:
        base_image = constants.JOB_PARAMETERS[job_kind]["base_image"]

    # Create Pod template spec.
    pod_template_spec = models.V1PodTemplateSpec(
        metadata=models.V1ObjectMeta(
            annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
        ),
        spec=models.V1PodSpec(
            containers=[
                models.V1Container(
                    name=constants.JOB_PARAMETERS[job_kind]["container"],
                    image=base_image,
                )
            ]
        ),
    )

    # If Training function is set, convert function to container execution script.
    if train_func is not None:
        # Check if function is callable.
        if not callable(train_func):
            raise ValueError(
                f"Training function must be callable, got function type: {type(train_func)}"
            )

        # Extract function implementation.
        func_code = inspect.getsource(train_func)

        # Function might be defined in some indented scope (e.g. in another function).
        # We need to dedent the function code.
        func_code = textwrap.dedent(func_code)

        # Wrap function code to execute it from the file. For example:
        # def train(parameters):
        #     print('Start Training...')
        # train({'lr': 0.01})
        if parameters is None:
            func_code = f"{func_code}\n{train_func.__name__}()\n"
        else:
            func_code = f"{func_code}\n{train_func.__name__}({parameters})\n"

        # Prepare execute script template.
        exec_script = textwrap.dedent(
            """
                program_path=$(mktemp -d)
                read -r -d '' SCRIPT << EOM\n
                {func_code}
                EOM
                printf "%s" \"$SCRIPT\" > \"$program_path/ephemeral_script.py\"
                python3 -u \"$program_path/ephemeral_script.py\""""
        )

        # Add function code to the execute script.
        exec_script = exec_script.format(func_code=func_code)

        # Install Python packages if that is required.
        if packages_to_install is not None:
            exec_script = (
                get_script_for_python_packages(packages_to_install, pip_index_url)
                + exec_script
            )

        # Add execution script to container arguments.
        pod_template_spec.spec.containers[0].command = ["bash", "-c"]
        pod_template_spec.spec.containers[0].args = [exec_script]

    return pod_template_spec


def get_tfjob_template(
    name: str,
    namespace: str,
    pod_template_spec: models.V1PodTemplateSpec,
    num_worker_replicas: Optional[int] = None,
    num_chief_replicas: Optional[int] = None,
    num_ps_replicas: Optional[int] = None,
):
    # Check if at least one replica is set.
    # TODO (andreyvelich): Remove this check once we have CEL validation.
    # Ref: https://github.com/kubeflow/training-operator/issues/1708
    if (
        num_worker_replicas is None
        and num_chief_replicas is None
        and num_ps_replicas is None
    ):
        raise ValueError("At least one replica for TFJob must be set")

    # Create TFJob template.
    tfjob = models.KubeflowOrgV1TFJob(
        api_version=constants.API_VERSION,
        kind=constants.TFJOB_KIND,
        metadata=models.V1ObjectMeta(name=name, namespace=namespace),
        spec=models.KubeflowOrgV1TFJobSpec(
            run_policy=models.KubeflowOrgV1RunPolicy(clean_pod_policy=None),
            tf_replica_specs={},
        ),
    )

    # Add Chief, PS, and Worker replicas to the TFJob.
    if num_chief_replicas is not None:
        tfjob.spec.tf_replica_specs[
            constants.REPLICA_TYPE_CHIEF
        ] = models.KubeflowOrgV1ReplicaSpec(
            replicas=num_chief_replicas,
            template=pod_template_spec,
        )

    if num_ps_replicas is not None:
        tfjob.spec.tf_replica_specs[
            constants.REPLICA_TYPE_PS
        ] = models.KubeflowOrgV1ReplicaSpec(
            replicas=num_ps_replicas,
            template=pod_template_spec,
        )

    if num_worker_replicas is not None:
        tfjob.spec.tf_replica_specs[
            constants.REPLICA_TYPE_WORKER
        ] = models.KubeflowOrgV1ReplicaSpec(
            replicas=num_worker_replicas,
            template=pod_template_spec,
        )

    return tfjob


def get_pytorchjob_template(
    name: str,
    namespace: str,
    pod_template_spec: models.V1PodTemplateSpec,
    num_worker_replicas: Optional[int] = None,
):
    # Check if at least one replica is set.
    # TODO (andreyvelich): Remove this check once we have CEL validation.
    # Ref: https://github.com/kubeflow/training-operator/issues/1708
    if num_worker_replicas is None:
        raise ValueError("At least one Worker replica for PyTorchJob must be set")

    # Create PyTorchJob template.
    pytorchjob = models.KubeflowOrgV1PyTorchJob(
        api_version=constants.API_VERSION,
        kind=constants.PYTORCHJOB_KIND,
        metadata=models.V1ObjectMeta(name=name, namespace=namespace),
        spec=models.KubeflowOrgV1PyTorchJobSpec(
            run_policy=models.KubeflowOrgV1RunPolicy(clean_pod_policy=None),
            pytorch_replica_specs={},
        ),
    )

    # Add Master and Worker replicas to the PyTorchJob.
    pytorchjob.spec.pytorch_replica_specs[
        constants.REPLICA_TYPE_MASTER
    ] = models.KubeflowOrgV1ReplicaSpec(
        replicas=1,
        template=pod_template_spec,
    )

    # If number of Worker replicas is 1, PyTorchJob uses only Master replica.
    if num_worker_replicas != 1:
        pytorchjob.spec.pytorch_replica_specs[
            constants.REPLICA_TYPE_WORKER
        ] = models.KubeflowOrgV1ReplicaSpec(
            replicas=num_worker_replicas,
            template=pod_template_spec,
        )

    return pytorchjob
