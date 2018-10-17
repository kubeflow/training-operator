"""Run an E2E test to verify invalid TFJobs are handled correctly.

If a TFJob is invalid it should be marked as failed with an appropriate
error message.
"""

import argparse
import logging
import json
import os
import re
import retrying

from kubernetes import client as k8s_client

from kubeflow.testing import test_helper
from kubeflow.testing import util
from py import ks_util
from py import test_util
from py import tf_job_client
from py import util as tf_operator_util

# One of the reasons we set so many retries and a random amount of wait
# between retries is because we have multiple tests running in parallel
# that are all modifying the same ksonnet app via ks. I think this can
# lead to failures.
@retrying.retry(stop_max_attempt_number=10, wait_random_min=1000,
                wait_random_max=10000)
def run_test(args, test_case):  # pylint: disable=too-many-branches,too-many-statements
  """Run a test."""
  util.load_kube_config()

  api_client = k8s_client.ApiClient()

  t = test_util.TestCase()
  t.class_name = "tfjob_test"
  namespace, name, env = ks_util.setup_ks_app(args)
  t.name = os.path.basename(name)

  try: # pylint: disable=too-many-nested-blocks
    util.run(["ks", "apply", env, "-c", args.component], cwd=args.app_dir)

    logging.info("Created job %s in namespaces %s", name, namespace)

    logging.info("Wait for conditions Failed")
    results = tf_job_client.wait_for_condition(
      api_client, namespace, name, ["Succeeded", "Failed"],
      status_callback=tf_job_client.log_status)

    logging.info("Final TFJob:\n %s", json.dumps(results, indent=2))

    # For v1alpha2 check for non-empty completionTime
    last_condition = results.get("status", {}).get("conditions", [])[-1]
    if last_condition.get("type", "").lower() != "failed":
      message = "Job {0} in namespace {1} did not fail; status {2}".format(
        name, namespace, results.get("status", {}))
      logging.error(message)
      test_case.add_failure_info(message)
      return

    pattern = ".*the spec is invalid.*"
    condition_message = last_condition.get("message", "")
    if not re.match(pattern, condition_message):
      message = "Condition message {0} did not match pattern {1}".format(
        condition_message, pattern)
      logging.error(message)
      test_case.add_failure_info(message)
  except tf_operator_util.JobTimeoutError as e:
    if e.job:
      spec = "Job:\n" + json.dumps(e.job, indent=2)
    else:
      spec = "JobTimeoutError did not contain job"
    message = ("Timeout waiting for {0} in namespace {1} to finish; ").format(
      name, namespace) + spec
    logging.exception(message)
    test_case.add_failure_info(message)
  except Exception as e:  # pylint: disable-msg=broad-except
    # TODO(jlewi): I'm observing flakes where the exception has message "status"
    # in an effort to try to nail down this exception we print out more
    # information about the exception.
    message = "There was a problem running the job; Exception {0}".format(e)
    logging.exception(message)
    test_case.add_failure_info(message)

def parse_args():
  """Parase arguments."""
  parser = argparse.ArgumentParser(description="Run a TFJob test.")

  parser.add_argument(
    "--app_dir",
    default=None,
    type=str,
    help="Directory containing the ksonnet app.")

  parser.add_argument(
    "--component",
    default="invalid-tfjob",
    type=str,
    help="The ksonnet component of the job to run.")

  parser.add_argument(
    "--params",
    default=None,
    type=str,
    help="Comma separated list of key value pairs to set on the component.")

  # parse the args and call whatever function was selected
  args, _ = parser.parse_known_args()

  return args

def test_invalid_job(test_case): # pylint: disable=redefined-outer-name
  args = parse_args()
  util.maybe_activate_service_account()

  run_test(args, test_case)

def main():
  test_case = test_helper.TestCase(
    name="test_invalid_job", test_func=test_invalid_job)
  test_suite = test_helper.init(
    name="test_invalid_job", test_cases=[test_case])
  test_suite.run()

if __name__ == "__main__":
  main()
