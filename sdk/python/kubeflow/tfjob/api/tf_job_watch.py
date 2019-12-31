# Copyright 2019 The Kubeflow Authors.
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
from table_logger import TableLogger

from kubeflow.tfjob.constants import constants
from kubeflow.tfjob.utils import utils

tbl = TableLogger(
  columns='NAME,STATE,TIME',
  colwidth={'NAME': 30, 'STATE':20, 'TIME':30},
  border=False)

@retrying.retry(wait_fixed=1000, stop_max_attempt_number=20)
def watch(name=None, namespace=None, timeout_seconds=600):
  """Watch the created or patched InferenceService in the specified namespace"""

  if namespace is None:
    namespace = utils.get_default_target_namespace()

  stream = k8s_watch.Watch().stream(
    client.CustomObjectsApi().list_namespaced_custom_object,
    constants.TFJOB_GROUP,
    constants.TFJOB_VERSION,
    namespace,
    constants.TFJOB_PLURAL,
    timeout_seconds=timeout_seconds)

  for event in stream:
    tfjob = event['object']
    tfjob_name = tfjob['metadata']['name']
    if name and name != tfjob_name:
      continue
    else:
      status = ''
      update_time = ''
      last_condition = tfjob.get('status', {}).get('conditions', [])[-1]
      status = last_condition.get('type', '')
      update_time = last_condition.get('lastTransitionTime', '')

      tbl(tfjob_name, status, update_time)

      if name == tfjob_name:
        if status == 'Succeeded' or status == 'Failed':
          break
