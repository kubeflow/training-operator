import json
import logging
from kubernetes import client as k8s_client
from kubeflow.testing import util
from py import ks_util
from py import tf_job_client

def run_tfjob_with_shutdown_policy(test_case, args, shutdown_policy):
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

  if shutdown_policy == "worker":
    tf_job_client.terminate_replicas(api_client, namespace, name, "worker", 1)
  else:
    tf_job_client.terminate_replicas(api_client, namespace, name, "chief", 1)
      
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

  logging.info("Waiting for job %s in namespaces %s to be deleted.", name,
               namespace)
  tf_job_client.wait_for_delete(
    api_client, namespace, name, args.tfjob_version, status_callback=tf_job_client.log_status)

  return True

# Tests launching a TFJob with a Chief replica. Terminate the chief replica, and
# verifies that the TFJob completes.
def test_shutdown_chief(test_case, args):
  return run_tfjob_with_shutdown_policy(test_case, args, "chief")

# Tests launching a TFJob with no Chief replicas. Terminate worker 0 (which becomes chief), and
# verifies that the TFJob completes.
def test_shutdown_worker0(test_case, args):
  return run_tfjob_with_shutdown_policy(test_case, args, "worker")
