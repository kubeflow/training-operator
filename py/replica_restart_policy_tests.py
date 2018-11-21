import json
import logging
from kubernetes import client as k8s_client
from kubeflow.testing import test_util, util
from py import ks_util
from py import test_runner
from py import tf_job_client

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
    api_client = k8s_client.ApiClient()

    # Setup the ksonnet app
    ks_util.setup_ks_app(self.app_dir, self.env, self.namespace, component,
                         self.params)

    # Create the TF job
    util.run(["ks", "apply", self.env, "-c", component], cwd=self.app_dir)
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

    if replica_restart_policy == "Always":
      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "PS", ["Running"])
      first_start_time = tf_job_client.get_start_time_by_index(
        api_client, self.namespace, self.name, "PS", 0)

      tf_job_client.terminate_replicas(api_client, self.namespace, self.name,
                                       "ps", 1, exit_code)

      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "PS", ["Running"])
      restart_time = tf_job_client.get_start_time_by_index(
        api_client, self.namespace, self.name, "PS", 0)

      logging.info("First start time: %s, restart time: %s",
                   str(first_start_time), str(restart_time))

      if restart_time <= first_start_time:
        self.failure = "Job {0} in namespace {1} with restart policy Always failed test".format(
          self.name, self.namespace)
        logging.error(self.failure)
        return

    elif replica_restart_policy == "OnFailure":
      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "PS", ["Running"])
      first_start_time = tf_job_client.get_start_time_by_index(
        api_client, self.namespace, self.name, "PS", 0)

      tf_job_client.terminate_replicas(api_client, self.namespace, self.name,
                                       "ps", 1, exit_code)

      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "PS", ["Running"])
      restart_time = tf_job_client.get_start_time_by_index(
        api_client, self.namespace, self.name, "PS", 0)

      logging.info("First start time: %s, restart time: %s",
                   str(first_start_time), str(restart_time))

      if restart_time <= first_start_time:
        self.failure = "Job {0} in namespace {1} with restart policy OnFailure \
          failed to restart the pod with exit_code 1".format(
          self.name, self.namespace)
        logging.error(self.failure)
        return

    elif replica_restart_policy == "Never":
      tf_job_client.terminate_replicas(api_client, self.namespace, self.name,
                                       "ps", 1, exit_code)
      tf_job_client.wait_for_replica_type_in_phases(api_client, self.namespace,
                                                    self.name, "PS", ["Failed"])

    elif replica_restart_policy == "ExitCode" and exit_code == 1:
      tf_job_client.terminate_replicas(api_client, self.namespace, self.name,
                                       "ps", 1, exit_code)
      tf_job_client.wait_for_replica_type_in_phases(api_client, self.namespace,
                                                    self.name, "PS", ["Failed"])

    else:
      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "PS", ["Running"])
      first_start_time = tf_job_client.get_start_time_by_index(
        api_client, self.namespace, self.name, "PS", 0)

      tf_job_client.terminate_replicas(api_client, self.namespace, self.name,
                                       "ps", 1, exit_code)

      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "PS", ["Running"])
      restart_time = tf_job_client.get_start_time_by_index(
        api_client, self.namespace, self.name, "PS", 0)

      logging.info("First start time: %s, restart time: %s",
                   str(first_start_time), str(restart_time))

      if restart_time <= first_start_time:
        self.failure = "Job {0} in namespace {1} with restart policy ExitCode \
          failed to restart the pod with exit_code 130".format(
          self.name, self.namespace)
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
  def test_restart_always(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_ALWAYS_COMPONENT_NAME + "_" + self.tfjob_version,
      "Always", 0)

  # Verify that the pod is restarted after failure.
  # We terminate PS with exit_code=1, and verify it is restarted.
  def test_restart_onfailure(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_ONFAILURE_COMPONENT_NAME + "_" +
      self.tfjob_version, "OnFailure", 1)

  # Verify that the pod is never restarted.
  # We terminate PS with exit_code=1, and verify that its phase becomes Failed.
  def test_restart_never(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_NEVER_COMPONENT_NAME + "_" + self.tfjob_version,
      "Never", 1)

  # Verify that the pod is not restarted after permanent error ( 1-127 ).
  # We terminate PS with exit_code=1, and verify its phase becomes Failed.
  def test_restart_exitcode_permanent_error(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_EXITCODE_COMPONENT_NAME + "_" + self.tfjob_version,
      "ExitCode", 1)

  # Verify that the pod is not restarted after permanent error ( 128-255 ).
  # We terminate PS with exit_code=128, and verify it is restarted.
  def test_restart_exitcode_retryable_error(self):
    return self.run_tfjob_with_replica_restart_policy(
      REPLICA_RESTART_POLICY_EXITCODE_COMPONENT_NAME + "_" + self.tfjob_version,
      "ExitCode", 130)


if __name__ == "__main__":
  test_runner.main(module=__name__)
