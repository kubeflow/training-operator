import json
import logging
from kubernetes import client as k8s_client
from kubeflow.testing import util
from py import ks_util
from py import tf_job_client

# Run a generic TFJob, wait for it to complete, and check for pod/service creation errors.
def run_simple_tfjob(test_case, args):
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

  # Check for creation failures.
  creation_failures = tf_job_client.get_creation_failures_from_tfjob(
    api_client, namespace, results)
  if creation_failures:
    # TODO(jlewi): Starting with
    # https://github.com/kubeflow/tf-operator/pull/646 the number of events
    # no longer seems to match the expected; it looks like maybe events
    # are being combined? For now we just log a warning rather than an
    # error.
    logging.warning(creation_failures)

  # Delete the TFJob.
  tf_job_client.delete_tf_job(api_client, namespace, name, version=args.tfjob_version)
  logging.info("Waiting for job %s in namespaces %s to be deleted.", name,
               namespace)
  tf_job_client.wait_for_delete(
    api_client, namespace, name, args.tfjob_version, status_callback=tf_job_client.log_status)

  return True
