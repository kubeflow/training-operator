import json
import logging

from kubeflow.testing import ks_util, test_util, util
from kubeflow.tf_operator import test_runner, tf_job_client
from kubeflow.tf_operator import util as tf_operator_util
from kubernetes import client as k8s_client

MASTER_IS_CHIEF_COMPONENT_NAME = "master_is_chief"
WORKER0_IS_CHIEF_COMPONENT_NAME = "worker0_is_chief"


class ShutdownPolicyTests(test_util.TestCase):

  def __init__(self, args):
    namespace, name, env = test_runner.parse_runtime_params(args)
    self.app_dir = args.app_dir
    self.env = env
    self.namespace = namespace
    self.tfjob_version = args.tfjob_version
    self.params = args.params
    super(ShutdownPolicyTests, self).__init__(
      class_name="ShutdownPolicyTests", name=name)

  def run_tfjob_with_shutdown_policy(self, component, shutdown_policy):
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

    if shutdown_policy == "worker":
      tf_job_client.terminate_replicas(api_client, self.namespace, self.name,
                                       "worker", 1)
    else:
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
      return

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

  # Tests launching a TFJob with a Chief replica. Terminate the chief replica, and
  # verifies that the TFJob completes.
  def test_shutdown_chief(self):
    return self.run_tfjob_with_shutdown_policy(
      MASTER_IS_CHIEF_COMPONENT_NAME + "_" + self.tfjob_version, "chief")

  # Tests launching a TFJob with no Chief replicas. Terminate worker 0 (which becomes chief), and
  # verifies that the TFJob completes.
  def test_shutdown_worker0(self):
    return self.run_tfjob_with_shutdown_policy(
      WORKER0_IS_CHIEF_COMPONENT_NAME + "_" + self.tfjob_version, "worker")


if __name__ == "__main__":
  test_runner.main(module=__name__)
