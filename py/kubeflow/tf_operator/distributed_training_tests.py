import json
import logging

from kubeflow.testing import ks_util, test_util, util
from kubeflow.tf_operator import test_runner, tf_job_client
from kubeflow.tf_operator import util as tf_operator_util
from kubernetes import client as k8s_client

TFJOB_COMPONENT_NAME = "distributed_training"


class DistributedTrainingJobTests(test_util.TestCase):

  def __init__(self, args):
    namespace, name, env = test_runner.parse_runtime_params(args)
    self.app_dir = args.app_dir
    self.env = env
    self.namespace = namespace
    self.tfjob_version = args.tfjob_version
    self.params = args.params
    super(DistributedTrainingJobTests, self).__init__(
      class_name="DistributedTrainingJobTests", name=name)

  # Run a distributed training TFJob, wait for it to complete, and check for pod/service
  # creation errors.
  def run_distributed_training_job(self, component):
    tf_operator_util.load_kube_config()
    api_client = k8s_client.ApiClient()

    # Setup the ksonnet app
    tf_operator_util.setup_ks_app(self.app_dir, self.env, self.namespace, component,
                         self.params)

    # Create the TF job
    ks_cmd = ks_util.get_ksonnet_cmd(self.app_dir)
    util.run([ks_cmd, "apply", self.env, "-c", component], cwd=self.app_dir)
    logging.info("Created job %s in namespaces %s", self.name, self.namespace)

    # Wait for the job to either be in Running state or a terminal state
    logging.info("Wait for conditions Running, Succeeded, or Failed")
    results = tf_job_client.wait_for_condition(
      api_client,
      self.namespace,
      self.name, ["Running", "Succeeded", "Failed"],
      version=self.tfjob_version,
      status_callback=tf_job_client.log_status)
    logging.info("Current TFJob:\n %s", json.dumps(results, indent=2))

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
      return

    # Check for creation failures.
    creation_failures = tf_job_client.get_creation_failures_from_tfjob(
      api_client, self.namespace, results)
    if creation_failures:
      logging.warning(creation_failures)

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

  # Run a distributed training TFJob, wait for it to complete, and check for pod/service
  # creation errors.
  def test_distributed_training_independent_worker(self):
    self.run_distributed_training_job(TFJOB_COMPONENT_NAME + "_" +
                                      self.tfjob_version)


if __name__ == "__main__":
  test_runner.main(module=__name__)
