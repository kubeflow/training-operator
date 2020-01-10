# Copyright 2019 kubeflow.org.
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

from kubeflow.tfjob.constants import constants

def is_running_in_k8s():
  return os.path.isdir('/var/run/secrets/kubernetes.io/')


def get_current_k8s_namespace():
  with open('/var/run/secrets/kubernetes.io/serviceaccount/namespace', 'r') as f:
    return f.readline()


def get_default_target_namespace():
  if not is_running_in_k8s():
    return 'default'
  return get_current_k8s_namespace()


def set_tfjob_namespace(tfjob):
  tfjob_namespace = tfjob.metadata.namespace
  namespace = tfjob_namespace or get_default_target_namespace()
  return namespace


def get_labels(name, master=False, replica_type=None, replica_index=None):
  """
  Get labels according to speficed flags.
  :param name: tfjob name
  :param master: if need include label 'job-role: master'.
  :param replica_type: User can specify one of 'worker, ps, chief to only' get one type pods.
  :param replica_index: Can specfy replica index to get one pod of TFJob.
  :return: Dict: Labels
  """
  labels = {
    constants.TFJOB_GROUP_LABEL: 'kubeflow.org',
    constants.TFJOB_CONTROLLER_LABEL: 'tf-operator',
    constants.TFJOB_NAME_LABEL: name,
  }

  if master:
    labels[constants.TFJOB_ROLE_LABEL] = 'master'

  if replica_type:
    labels[constants.TFJOB_TYPE_LABEL] = str.lower(replica_type)

  if replica_index:
    labels[constants.TFJOB_INDEX_LABEL] = replica_index

  return labels


def to_selector(labels):
  """
  Transfer Labels to selector.
  """
  parts = []
  for key in labels.keys():
    parts.append("{0}={1}".format(key, labels[key]))

  return ",".join(parts)
