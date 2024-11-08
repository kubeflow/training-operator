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

import json
import os

from kubeflow.training.constants import constants
from kubernetes import config


def is_running_in_k8s():
    return os.path.isdir("/var/run/secrets/kubernetes.io/")


def get_default_target_namespace():
    if not is_running_in_k8s():
        try:
            _, current_context = config.list_kube_config_contexts()
            return current_context["context"]["namespace"]
        except Exception:
            return constants.DEFAULT_NAMESPACE
    with open("/var/run/secrets/kubernetes.io/serviceaccount/namespace", "r") as f:
        return f.readline()


class FakeResponse:
    """Fake object of RESTResponse to deserialize
    Ref) https://github.com/kubeflow/katib/pull/1630#discussion_r697877815
    Ref) https://github.com/kubernetes-client/python/issues/977#issuecomment-592030030
    """

    def __init__(self, obj):
        self.data = json.dumps(obj)
