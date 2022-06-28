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

from kubeflow.training.constants import constants


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
        constants.JOB_GROUP_LABEL: 'kubeflow.org',
        constants.JOB_NAME_LABEL: name,
    }
    if master:
        labels[constants.JOB_ROLE_LABEL] = 'master'

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
