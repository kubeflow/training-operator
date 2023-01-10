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

from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container

from kubeflow.training import TrainingClient
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1XGBoostJob
from kubeflow.training import KubeflowOrgV1XGBoostJobSpec
from kubeflow.training import V1RunPolicy
from kubeflow.training.constants import constants

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
SDK_TEST_NAMESPACE = "default"
JOB_NAME = "xgboostjob-iris-ci-test"


def test_sdk_e2e():
    container = V1Container(
        name="xgboost",
        image="docker.io/merlintang/xgboost-dist-iris:1.1",
        args=[
            "--job_type=Train",
            "--xgboost_parameter=objective:multi:softprob,num_class:3",
            "--n_estimators=10",
            "--learning_rate=0.1",
            "--model_path=/tmp/xgboost-model",
            "--model_storage_type=local",
        ],
    )

    master = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[container])),
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(spec=V1PodSpec(containers=[container])),
    )

    xgboostjob = KubeflowOrgV1XGBoostJob(
        api_version="kubeflow.org/v1",
        kind="XGBoostJob",
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=SDK_TEST_NAMESPACE),
        spec=KubeflowOrgV1XGBoostJobSpec(
            run_policy=V1RunPolicy(clean_pod_policy="None",),
            xgb_replica_specs={"Master": master, "Worker": worker},
        ),
    )

    TRAINING_CLIENT.create_xgboostjob(xgboostjob, SDK_TEST_NAMESPACE)

    TRAINING_CLIENT.wait_for_job_conditions(
        JOB_NAME, SDK_TEST_NAMESPACE, constants.XGBOOSTJOB_KIND
    )

    TRAINING_CLIENT.get_job_logs(JOB_NAME, SDK_TEST_NAMESPACE)

    TRAINING_CLIENT.delete_xgboostjob(JOB_NAME, SDK_TEST_NAMESPACE)
