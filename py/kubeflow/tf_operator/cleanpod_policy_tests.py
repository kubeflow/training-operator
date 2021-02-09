import json
import logging

from kubeflow.testing import ks_util, test_util, util
from kubeflow.tf_operator import k8s_util, test_runner, tf_job_client
from kubeflow.tf_operator import util as tf_operator_util
from kubernetes import client as k8s_client

CLEANPOD_ALL_COMPONENT_NAME = "clean_pod_all"
CLEANPOD_RUNNING_COMPONENT_NAME = "clean_pod_running"
CLEANPOD_NONE_COMPONENT_NAME = "clean_pod_none"


class CleanPodPolicyTests(test_util.TestCase):

  def __init__(self, args):
    namespace, name, env = test_runner.parse_runtime_params(args)
    self.app_dir = args.app_dir
    self.env = env
    self.namespace = namespace
    self.tfjob_version = args.tfjob_version
    self.params = args.params
    super(CleanPodPolicyTests, self).__init__(
      class_name="CleanPodPolicyTests", name=name)

  def run_tfjob_with_cleanpod_policy(self, component, clean_pod_policy):
    tf_operator_util.load_kube_config()
    api_client = k8s_client.ApiClient()

    # Setup the ksonnet app
    ks_cmd = ks_util.get_ksonnet_cmd(self.app_dir)
    tf_operator_util.setup_ks_app(self.app_dir, self.env, self.namespace, component,
                         self.params)

    # Create the TF job
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

    # All pods are deleted.
    if clean_pod_policy == "All":
      pod_labels = tf_job_client.get_labels(self.name)
      pod_selector = tf_job_client.to_selector(pod_labels)
      k8s_util.wait_for_pods_to_be_deleted(api_client, self.namespace,
                                           pod_selector)
    # Only running pods (PS) are deleted, completed pods are not.
    elif clean_pod_policy == "Running":
      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "Chief", ["Succeeded"])
      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "Worker", ["Succeeded"])
      pod_labels = tf_job_client.get_labels(self.name, "PS")
      pod_selector = tf_job_client.to_selector(pod_labels)
      k8s_util.wait_for_pods_to_be_deleted(api_client, self.namespace,
                                           pod_selector)
    # No pods are deleted.
    elif clean_pod_policy == "None":
      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "Chief", ["Succeeded"])
      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "Worker", ["Succeeded"])
      tf_job_client.wait_for_replica_type_in_phases(
        api_client, self.namespace, self.name, "PS", ["Running"])

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

  # Verify that all pods are deleted when the job completes.
  def test_cleanpod_all(self):
    return self.run_tfjob_with_cleanpod_policy(
      CLEANPOD_ALL_COMPONENT_NAME + "_" + self.tfjob_version, "All")

  # Verify that running pods are deleted when the job completes.
  def test_cleanpod_running(self):
    return self.run_tfjob_with_cleanpod_policy(
      CLEANPOD_RUNNING_COMPONENT_NAME + "_" + self.tfjob_version, "Running")

  # Verify that none of the pods are deleted when the job completes.
  def test_cleanpod_none(self):
    return self.run_tfjob_with_cleanpod_policy(
      CLEANPOD_NONE_COMPONENT_NAME + "_" + self.tfjob_version, "None")


if __name__ == "__main__":
  test_runner.main(module=__name__)
