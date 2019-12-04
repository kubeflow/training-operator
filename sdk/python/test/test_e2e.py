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

import time
import os
from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container

from kubeflow.tfjob import V1ReplicaSpec
from kubeflow.tfjob import V1TFJob
from kubeflow.tfjob import V1TFJobSpec
from kubeflow.tfjob import TFJobClient


TFJOB_CLIENT = TFJobClient(config_file=os.getenv('KUBECONFIG'))

def wait_for_tfjob_ready(name, namespace='default',
                        timeout_seconds=600):
  for _ in range(round(timeout_seconds/10)):
    time.sleep(10)
    tfjob = TFJOB_CLIENT.get(name, namespace=namespace)

    last_condition = tfjob.get("status", {}).get("conditions", [])[-1]
    last_status = last_condition.get("type", "").lower()

    if last_status == "succeeded":
      return
    elif last_status == "failed":
      raise RuntimeError("The TFJob is failed.")
    else:
      continue

    raise RuntimeError("Timeout to finish the TFJob.")

def test_sdk_e2e():

  container = V1Container(
  name="tensorflow",
  image="gcr.io/kubeflow-ci/tf-mnist-with-summaries:1.0",
  command=[
      "python",
      "/var/tf_mnist/mnist_with_summaries.py",
      "--log_dir=/train/logs", "--learning_rate=0.01",
      "--batch_size=150"
      ]
  )

  worker = V1ReplicaSpec(
    replicas=1,
    restart_policy="Never",
    template=V1PodTemplateSpec(
      spec=V1PodSpec(
        containers=[container]
        )
    )
  )

  tfjob = V1TFJob(
    api_version="kubeflow.org/v1",
    kind="TFJob",
    metadata=V1ObjectMeta(name="mnist-ci-test", namespace='default'),
    spec=V1TFJobSpec(
      clean_pod_policy="None",
      tf_replica_specs={"Worker": worker}
    )
  )


  TFJOB_CLIENT.create(tfjob)
  wait_for_tfjob_ready("mnist-ci-test")

  TFJOB_CLIENT.delete("mnist-ci-test", namespace='default')
