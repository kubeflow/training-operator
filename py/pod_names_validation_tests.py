"""Tests for pod_names_validation."""

import json
import logging
from kubernetes import client as k8s_client
from kubeflow.testing import ks_util, test_util, util
from py import test_runner
from py import tf_job_client

COMPONENT_NAME = "pod_names_validation"

def extract_job_specs(replica_specs):
  specs = dict()
  for job_type in replica_specs:
    specs[job_type.encode("ascii").lower()] = replica_specs.get(job_type, {}).get("replicas", 0)
  return specs

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

    job_specs = extract_job_specs(results.get("spec", {})
                                         .get("tfReplicaSpecs", {}))
    expected_pod_names = []
    for job_type, replica in job_specs:
      logging.info("replica type = %s", type(replica))
      for i in xrange(replica):
        expected_pod_names.append("{name}-{replica}-{index}",
          name=self.name, replica=job_type, index=i)
    expected_pod_names = tuple(expected_pod_names)
    pod_names = tf_job_client.get_pod_names(api_client,
                                            self.namespace,
                                            self.name)

    logging.info("Expected = %s", str(expected_pod_names))
    logging.info("Actual = %s", str(pod_names))

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
