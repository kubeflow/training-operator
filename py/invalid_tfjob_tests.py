import json
import logging
import re
from kubernetes import client as k8s_client
from kubeflow.testing import ks_util
from kubeflow.testing import test_util, util
from py import test_runner
from py import tf_job_client

INVALID_TFJOB_COMPONENT_NAME = "invalid_tfjob"


class InvalidTfJobTests(test_util.TestCase):

  def __init__(self, args):
    namespace, name, env = test_runner.parse_runtime_params(args)
    self.app_dir = args.app_dir
    self.env = env
    self.namespace = namespace
    self.tfjob_version = args.tfjob_version
    self.params = args.params
    super(InvalidTfJobTests, self).__init__(
      class_name="InvalidTfJobTests", name=name)

  def test_invalid_tfjob_spec(self):
    api_client = k8s_client.ApiClient()
    component = INVALID_TFJOB_COMPONENT_NAME + "_" + self.tfjob_version

    # Setup the ksonnet app
    ks_util.setup_ks_app(self.app_dir, self.env, self.namespace, component,
                         self.params)

    # Create the TF job
    util.run(["ks", "apply", self.env, "-c", component], cwd=self.app_dir)
    logging.info("Created job %s in namespaces %s", self.name, self.namespace)

    logging.info("Wait for conditions Failed")
    results = tf_job_client.wait_for_condition(
      api_client,
      self.namespace,
      self.name, ["Failed"],
      version=self.tfjob_version,
      status_callback=tf_job_client.log_status)

    logging.info("Final TFJob:\n %s", json.dumps(results, indent=2))

    # For v1alpha2 check for non-empty completionTime
    last_condition = results.get("status", {}).get("conditions", [])[-1]
    if last_condition.get("type", "").lower() != "failed":
      self.failure = "Job {0} in namespace {1} did not fail; status {2}".format(
        self.name, self.namespace, results.get("status", {}))
      logging.error(self.failure)
      return

    pattern = ".*the spec is invalid.*"
    condition_message = last_condition.get("message", "")
    if not re.match(pattern, condition_message):
      self.failure = "Condition message {0} did not match pattern {1}".format(
        condition_message, pattern)
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


if __name__ == "__main__":
  test_runner.main(module=__name__)
