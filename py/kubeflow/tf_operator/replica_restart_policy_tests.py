import json
import logging

from kubeflow.testing import ks_util, test_util, util
from kubeflow.tf_operator import test_runner, tf_job_client
from kubeflow.tf_operator import util as tf_operator_util
from kubernetes import client as k8s_client

REPLICA_RESTART_POLICY_ALWAYS_COMPONENT_NAME = "replica_restart_policy_always"
REPLICA_RESTART_POLICY_ONFAILURE_COMPONENT_NAME = "replica_restart_policy_onfailure"
REPLICA_RESTART_POLICY_NEVER_COMPONENT_NAME = "replica_restart_policy_never"
REPLICA_RESTART_POLICY_EXITCODE_COMPONENT_NAME = "replica_restart_policy_exitcode"


class ReplicaRestartPolicyTests(test_util.TestCase):

  def __init__(self, args):
    namespace, name, env = test_runner.parse_runtime_params(args)
    self.app_dir = args.app_dir
    self.env = env
    self.namespace = namespace
    self.tfjob_version = args.tfjob_version
    self.params = args.params
    super(ReplicaRestartPolicyTests, self).__init__(
      class_name="ReplicaRestartPolicyTests", name=name)

  def run_tfjob_with_replica_restart_policy(self, component,
                                            replica_restart_policy, exit_code):
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

    if replica_restart_policy == "Always" and exit_code == 0:
      res = tf_job_client.terminate_and_verify_start_time(
        api_client, self.namespace, self.name, "ps", 0, exit_code, True)

    elif replica_restart_policy == "Always" and exit_code == 1:
      res = tf_job_client.terminate_and_verify_start_time(
        api_client, self.namespace, self.name, "ps", 0, exit_code, True)

    elif replica_restart_policy == "OnFailure" and exit_code == 1:
      res = tf_job_client.terminate_and_verify_start_time(
        api_client, self.namespace, self.name, "ps", 0, exit_code, True)

    elif replica_restart_policy == "OnFailure" and exit_code == 0:
      res = tf_job_client.terminate_and_verify_start_time(
        api_client, self.namespace, self.name, "ps", 0, exit_code, False)

    elif replica_restart_policy == "Never" and exit_code == 1:
      res = tf_job_client.terminate_and_verify_start_time(
        api_client, self.namespace, self.name, "ps", 0, exit_code, False)

    elif replica_restart_policy == "Never" and exit_code == 0:
      res = tf_job_client.terminate_and_verify_start_time(
        api_client, self.namespace, self.name, "ps", 0, exit_code, False)

    elif replica_restart_policy == "ExitCode" and exit_code == 1:
      res = tf_job_client.terminate_and_verify_start_time(
        api_client, self.namespace, self.name, "ps", 0, exit_code, False)

    else:
      res = tf_job_client.terminate_and_verify_start_time(
        api_client, self.namespace, self.name, "ps", 0, exit_code, True)

    if res is False:
      self.failure = "Job {0} in namespace {1} with restart policy {2} failed test \
        with exit_code {3}".format(self.name, self.namespace,
                                   replica_restart_policy, exit_code)
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

  # Verify that the pod is restarted even after the container exits with success.
  # We terminate PS with exit_code=0, and verify it is restarted.
  def test_restart_always_exit_code_0(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_ALWAYS_COMPONENT_NAME + "_" + self.tfjob_version,
      "Always", 0)

  # Verify that the pod is restarted after the container exits with 1.
  # We terminate PS with exit_code=1, and verify it is restarted.
  def test_restart_always_exit_code_1(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_ALWAYS_COMPONENT_NAME + "_" + self.tfjob_version,
      "Always", 1)

  # Verify that the pod is restarted after failure.
  # We terminate PS with exit_code=1, and verify it is restarted.
  def test_restart_onfailure_exit_code_1(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_ONFAILURE_COMPONENT_NAME + "_" +
      self.tfjob_version, "OnFailure", 1)

  # Verify that the pod is restarted after failure.
  # We terminate PS with exit_code=0, and verify it is not restarted.
  def test_restart_onfailure_exit_code_0(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_ONFAILURE_COMPONENT_NAME + "_" +
      self.tfjob_version, "OnFailure", 0)

  # Verify that the pod is never restarted.
  # We terminate PS with exit_code=1, and verify it is not restarted.
  def test_restart_never_exit_code_1(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_NEVER_COMPONENT_NAME + "_" + self.tfjob_version,
      "Never", 1)

  # Verify that the pod is never restarted.
  # We terminate PS with exit_code=0, and verify it is not restarted.
  def test_restart_never_exit_code_0(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_NEVER_COMPONENT_NAME + "_" + self.tfjob_version,
      "Never", 0)

  # Verify that the pod is not restarted after permanent error ( 1-127 ).
  # We terminate PS with exit_code=1, and verify its phase becomes Failed.
  def test_restart_exitcode_permanent_error(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_EXITCODE_COMPONENT_NAME + "_" + self.tfjob_version,
      "ExitCode", 1)

  # Verify that the pod is not restarted after retryable error.
  # We terminate PS with exit_code=130, and verify it is restarted.
  def test_restart_exitcode_retryable_error(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_EXITCODE_COMPONENT_NAME + "_" + self.tfjob_version,
      "ExitCode", 130)


if __name__ == "__main__":
  test_runner.main(module=__name__)
