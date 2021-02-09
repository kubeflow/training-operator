"""Tests for pod_names_validation. This test takes app manifest and validate instantiated
pod names are in the format POD_NAME_FORMAT mentioned.
"""

import json
import logging

from kubeflow.testing import ks_util, test_util, util
from kubeflow.tf_operator import test_runner, tf_job_client
from kubeflow.tf_operator import util as tf_operator_util
from kubernetes import client as k8s_client

COMPONENT_NAME = "pod_names_validation"
POD_NAME_FORMAT = "{name}-{replica}-{index}"


def extract_job_specs(replica_specs):
  """Extract tf job specs from tfReplicaSpecs.

  Args:
    replica_specs: A dictionary having information of tfReplicaSpecs from manifest.
    returns: Dictionary. Key is tf job type and value is number of replicas.
  """
  specs = dict()
  for job_type in replica_specs:
    specs[job_type.lower()] = int(
      replica_specs.get(job_type, {}).get("replicas", 0))
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
    self.failure = None
    logging.info("env = %s", str(self.env))
    logging.info("params = %s", str(self.params))
    super(PodNamesValidationTest, self).__init__(
      class_name="PodNamesValidationTest", name=name)

  def test_pod_names(self):
    tf_operator_util.load_kube_config()
    api_client = k8s_client.ApiClient()
    component = COMPONENT_NAME + "_" + self.tfjob_version
    ks_cmd = ks_util.get_ksonnet_cmd(self.app_dir)
    tf_operator_util.setup_ks_app(self.app_dir, self.env, self.namespace, component,
                         self.params)
    util.run([ks_cmd, "apply", self.env, "-c", component], cwd=self.app_dir)
    logging.info("Created job %s in namespaces %s", self.name, self.namespace)
    logging.info("Wait for conditions Running, Succeeded, or Failed")
    results = tf_job_client.wait_for_condition(
      api_client,
      self.namespace,
      self.name, ["Running", "Succeeded", "Failed"],
      version=self.tfjob_version,
      status_callback=tf_job_client.log_status)
    logging.info("Current TFJob:\n %s", json.dumps(results, indent=2))

    job_specs = extract_job_specs(
      results.get("spec", {}).get("tfReplicaSpecs", {}))
    expected_pod_names = []
    for replica_type, replica_num in job_specs.items():
      logging.info("job_type = %s, replica = %s", replica_type, replica_num)
      for i in range(replica_num):
        expected_pod_names.append(
          POD_NAME_FORMAT.format(name=self.name, replica=replica_type, index=i))
    expected_pod_names = set(expected_pod_names)
    actual_pod_names = tf_job_client.get_pod_names(api_client, self.namespace,
                                                   self.name)

    # We are not able to guarantee pods selected with default namespace and job
    # name are only for this test run only. Therefore we only do partial check,
    # e.g. make sure expected set of pod names are in the selected pod names.
    if not (expected_pod_names & actual_pod_names) == expected_pod_names:
      msg = "Actual pod names doesn't match. Expected: {0} Actual: {1}".format(
        str(expected_pod_names), str(actual_pod_names))
      logging.error(msg)
      raise RuntimeError(msg)

    tf_job_client.terminate_replicas(api_client, self.namespace, self.name,
                                     "chief", 1)
    # Wait for the job to complete.
    logging.info("Waiting for job to finish.")
    results = tf_job_client.wait_for_job(
      api_client,
      self.namespace,
      self.name,
      self.tfjob_version,
      status_callback=tf_job_client.log_status)
    logging.info("Final TFJob:\n %s", json.dumps(results, indent=2))

    if not tf_job_client.job_succeeded(results):
      self.failure = "Job {0} in namespace {1} in status {2}".format(
        self.name, self.namespace, results.get("status", {}))
      logging.error(self.failure)

    # Delete the TFJob.
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
