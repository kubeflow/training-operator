"""Tests for pod_names_validation."""

import json
import logging
from kubernetes import client as k8s_client
from kubeflow.testing import ks_util, test_util, util
from py import test_runner
from py import tf_job_client

COMPONENT_NAME = "pod_names_validation"

class PodNamesValidationTest(test_util.TestCase):

  def __init__(self, args):
    namespace, name, env = test_runner.parse_runtime_params(args)
    self.app_dir = args.app_dir
    self.cluster_name = args.cluster
    self.env = env
    self.namespace = namespace
    self.tfjob_version = args.tfjob_version
    self.params = args.params
    super(PodNamesValidationTest, self).__init__(
        class_name="PodNamesValidationTest", name=name)

  def test_pod_names(self):
    api_client = k8s_client.ApiClient()
    # masterHost = api_client.configuration.host
    component = COMPONENT_NAME + "_" + self.tfjob_version

    ks_util.setup_ks_app(self.app_dir, self.env, self.namespace, component,
                         self.params)
    util.run(["ks", "apply", self.env, "-c", component], cwd=self.app_dir)
    logging.info("Created job %s in namespaces %s", self.name, self.namespace)
    logging.info("Wait for conditions Running, Succeeded, or Failed")
    results = tf_job_client.wait_for_condition(
      api_client,
      self.namespace,
      self.name, ["Running", "Succeeded", "Failed"],
      version=self.tfjob_version,
      status_callback=tf_job_client.log_status)
    logging.info("Current TFJob:\n %s", json.dumps(results, indent=2))

    tf_job_client.get_jobs(api_client, self.cluster_name, self.namespace)

    tf_job_client.delete_tf_job(
      api_client, self.namespace, self.name, version=self.tfjob_version)
    logging.info("Waiting for job %s in namespaces %s to be deleted.",
                 self.name, self.namespace)
    tf_job_client.wait_for_delete(
      api_client,
      self.namespace,
      self.name,
      self.tfjob_version,
      status_callback=tf_job_client.log_status)

if __name__ == '__main__':
  test_runner.main(module=__name__)
