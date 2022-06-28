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
from kubernetes.client import V1ContainerPort

from kubeflow.training import MXJobClient
from kubeflow.training import V1ReplicaSpec
from kubeflow.training import KubeflowOrgV1MXJob
from kubeflow.training import KubeflowOrgV1MXJobSpec
from kubeflow.training import V1RunPolicy

MX_CLIENT = MXJobClient(config_file=os.getenv('KUBECONFIG', '~/.kube/config'))
SDK_TEST_NAMESPACE = 'default'


def test_sdk_e2e():
    worker_container = V1Container(
        name="mxnet",
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        command=["/usr/local/bin/python3"],
        args=["incubator-mxnet/example/image-classification/train_mnist.py",
              "--num-epochs", "5",
              "--num-examples","1000",
              "--kv-store", "dist_sync"],
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")]
    )

    server_container = V1Container(
        name="mxnet",
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")]
    )

    scheduler_container = V1Container(
        name="mxnet",
        image="docker.io/johnugeorge/mxnet:1.9.1_cpu_py3",
        ports=[V1ContainerPort(container_port=9991, name="mxjob-port")]
    )

    worker = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            spec=V1PodSpec(
                containers=[worker_container]
            )
        )
    )

    server = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            spec=V1PodSpec(
                containers=[server_container]
            )
        )
    )

    scheduler = V1ReplicaSpec(
        replicas=1,
        restart_policy="Never",
        template=V1PodTemplateSpec(
            spec=V1PodSpec(
                containers=[scheduler_container]
            )
        )
    )

    mxjob = KubeflowOrgV1MXJob(
        api_version="kubeflow.org/v1",
        kind="MXJob",
        metadata=V1ObjectMeta(name="mxjob-mnist-ci-test", namespace=SDK_TEST_NAMESPACE),
        spec=KubeflowOrgV1MXJobSpec(
            job_mode="MXTrain",
            run_policy=V1RunPolicy(
                clean_pod_policy="None",
            ),
            mx_replica_specs={"Scheduler": scheduler,
                                "Server": server,
                                   "Worker": worker}
        )
    )

    MX_CLIENT.create(mxjob)

    MX_CLIENT.wait_for_job("mxjob-mnist-ci-test", namespace=SDK_TEST_NAMESPACE)
    if not MX_CLIENT.is_job_succeeded("mxjob-mnist-ci-test",
                                           namespace=SDK_TEST_NAMESPACE):
        raise RuntimeError("The MXJob is not succeeded.")

    MX_CLIENT.get_logs("mxjob-mnist-ci-test", namespace=SDK_TEST_NAMESPACE, master=False)

    MX_CLIENT.delete("mxjob-mnist-ci-test", namespace=SDK_TEST_NAMESPACE)
