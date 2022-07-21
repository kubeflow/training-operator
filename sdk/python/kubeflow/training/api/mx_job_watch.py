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

import retrying
from kubernetes import client
from kubernetes import watch as k8s_watch

from kubeflow.training.constants import constants
from kubeflow.training.utils import utils

tbl = utils.TableLogger(
    header="{:<30.30} {:<20.20} {:<30.30}".format('NAME', 'STATE', 'TIME'),
    column_format="{:<30.30} {:<20.20} {:<30.30}")


@retrying.retry(wait_fixed=1000, stop_max_attempt_number=20)
def watch(name=None, namespace=None, timeout_seconds=600):
    """Watch the created or patched InferenceService in the specified namespace"""

    if namespace is None:
        namespace = utils.get_default_target_namespace()

    stream = k8s_watch.Watch().stream(
        client.CustomObjectsApi().list_namespaced_custom_object,
        constants.MXJOB_GROUP,
        constants.MXJOB_VERSION,
        namespace,
        constants.MXJOB_PLURAL,
        timeout_seconds=timeout_seconds)

    for event in stream:
        mxjob = event['object']
        mxjob_name = mxjob['metadata']['name']
        if name and name != mxjob_name:
            continue
        else:
            status = ''
            update_time = ''
            last_condition = mxjob.get('status', {}).get('conditions', [{}])[-1]
            status = last_condition.get('type', '')
            update_time = last_condition.get('lastTransitionTime', '')

            tbl(mxjob_name, status, update_time)

            if name == mxjob_name:
                if status in [constants.JOB_STATUS_SUCCEEDED, constants.JOB_STATUS_FAILED]:
                    break
