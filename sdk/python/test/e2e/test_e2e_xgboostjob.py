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

from test.e2e.utils import verify_job_e2e

logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)

TRAINING_CLIENT = TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))
JOB_NAME = "xgboostjob-iris-ci-test"
JOB_NAMESPACE = "default"
CONTAINER_NAME = "xgboost"


def test_sdk_e2e():
    container = V1Container(
        name=CONTAINER_NAME,
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
        metadata=V1ObjectMeta(name=JOB_NAME, namespace=JOB_NAMESPACE),
        spec=KubeflowOrgV1XGBoostJobSpec(
            run_policy=V1RunPolicy(clean_pod_policy="None",),
            xgb_replica_specs={"Master": master, "Worker": worker},
        ),
    )

    TRAINING_CLIENT.create_xgboostjob(xgboostjob, JOB_NAMESPACE)
    logging.info(f"List of created {constants.XGBOOSTJOB_KIND}s")
    logging.info(TRAINING_CLIENT.list_xgboostjobs(JOB_NAMESPACE))

    verify_job_e2e(
        TRAINING_CLIENT,
        JOB_NAME,
        JOB_NAMESPACE,
        constants.XGBOOSTJOB_KIND,
        CONTAINER_NAME,
    )

    TRAINING_CLIENT.delete_xgboostjob(JOB_NAME, JOB_NAMESPACE)
