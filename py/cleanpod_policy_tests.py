import json
import logging
from kubernetes import client as k8s_client
from kubeflow.testing import util
from py import k8s_util
from py import ks_util
from py import tf_job_client

def run_tfjob_with_cleanpod_policy(test_case, args, clean_pod_policy):
  api_client = k8s_client.ApiClient()
  namespace, name, env = ks_util.setup_ks_app(args)

  # Create the TF job
  util.run(["ks", "apply", env, "-c", args.component], cwd=args.app_dir)
  logging.info("Created job %s in namespaces %s", name, namespace)

  # Wait for the job to either be in Running state or a terminal state
  logging.info("Wait for conditions Running, Succeeded, or Failed")
  results = tf_job_client.wait_for_condition(
    api_client, namespace, name, ["Running", "Succeeded", "Failed"],
    status_callback=tf_job_client.log_status)
  logging.info("Current TFJob:\n %s", json.dumps(results, indent=2))

  # Wait for the job to complete.
  logging.info("Waiting for job to finish.")
  results = tf_job_client.wait_for_job(
    api_client, namespace, name, args.tfjob_version,
    status_callback=tf_job_client.log_status)
  logging.info("Final TFJob:\n %s", json.dumps(results, indent=2))

  if not tf_job_client.job_succeeded(results):
    test_case.failure = "Job {0} in namespace {1} in status {2}".format(
      name, namespace, results.get("status", {}))
    logging.error(test_case.failure)
    return False

  # All pods are deleted.
  if clean_pod_policy == "All":
    pod_labels = tf_job_client.get_labels_v1alpha2(name)
    pod_selector = tf_job_client.to_selector(pod_labels)
    k8s_util.wait_for_pods_to_be_deleted(api_client, namespace, pod_selector)
  # Only running pods (PS) are deleted, completed pods are not.
  elif clean_pod_policy == "Running":
    tf_job_client.wait_for_replica_type_in_phases(api_client, namespace,
                                                  name, "Chief", ["Completed"])
    tf_job_client.wait_for_replica_type_in_phases(api_client, namespace, name,
                                                  "Worker", ["Completed"])
    pod_labels = tf_job_client.get_labels_v1alpha2(name, "PS")
    pod_selector = tf_job_client.to_selector(pod_labels)
    k8s_util.wait_for_pods_to_be_deleted(api_client, namespace, pod_selector)
  # No pods are deleted.
  elif clean_pod_policy == "None":
    tf_job_client.wait_for_replica_type_in_phases(api_client, namespace,
                                                  name, "Chief", ["Completed"])
    tf_job_client.wait_for_replica_type_in_phases(api_client, namespace,
                                                  name, "Worker", ["Completed"])
    tf_job_client.wait_for_replica_type_in_phases(api_client, namespace,
                                                  name, "PS", ["Running"])

  # Delete the TFJob.
  tf_job_client.delete_tf_job(api_client, namespace, name, version=args.tfjob_version)
  logging.info("Waiting for job %s in namespaces %s to be deleted.", name,
               namespace)
  tf_job_client.wait_for_delete(
    api_client, namespace, name, args.tfjob_version, status_callback=tf_job_client.log_status)

  return True

# Verify that all pods are deleted when the job completes.
def test_cleanpod_all(test_case, args):
  return run_tfjob_with_cleanpod_policy(test_case, args, "All")

# Verify that running pods are deleted when the job completes.
def test_cleanpod_running(test_case, args):
  return run_tfjob_with_cleanpod_policy(test_case, args, "Running")

# Verify that none of the pods are deleted when the job completes.
def test_cleanpod_none(test_case, args):
  return run_tfjob_with_cleanpod_policy(test_case, args, "None")
