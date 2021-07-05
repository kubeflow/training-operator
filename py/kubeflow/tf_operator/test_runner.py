"""Test runner runs a TFJob test."""

import argparse
import inspect
import json
import logging
import time
import uuid
from importlib import import_module

import retrying
from kubeflow.testing import test_util, util
from kubeflow.tf_operator import util as tf_operator_util


# One of the reasons we set so many retries and a random amount of wait
# between retries is because we have multiple tests running in parallel
# that are all modifying the same ksonnet app via ks. I think this can
# lead to failures.
@retrying.retry(
  stop_max_attempt_number=10, wait_random_min=1000, wait_random_max=10000)
def run_test(test_case, test_func, args):  # pylint: disable=too-many-branches,too-many-statements
  """Run a test."""
  util.load_kube_config()

  start = time.time()

  try:  # pylint: disable=too-many-nested-blocks
    # We repeat the test multiple times.
    # This ensures that if we delete the job we can create a new job with the
    # same name.

    num_trials = args.num_trials
    logging.info("tfjob_version=%s", args.tfjob_version)

    for trial in range(num_trials):
      logging.info("Trial %s", trial)
      test_func()

    # TODO(jlewi):
    #  Here are some validation checks to run:
    #  1. Check that all resources are garbage collected.
    # TODO(jlewi): Add an option to add chaos and randomly kill various resources?
    # TODO(jlewi): Are there other generic validation checks we should
    # run.
  except tf_operator_util.JobTimeoutError as e:
    if e.job:
      spec = "Job:\n" + json.dumps(e.job, indent=2)
    else:
      spec = "JobTimeoutError did not contain job"
    test_case.failure = "Timeout waiting for job to finish: " + spec
    logging.exception(test_case.failure)
  except Exception as e:  # pylint: disable-msg=broad-except
    # TODO(jlewi): I'm observing flakes where the exception has message "status"
    # in an effort to try to nail down this exception we print out more
    # information about the exception.
    logging.exception("There was a problem running the job; Exception %s", e)
    # We want to catch all exceptions because we want the test as failed.
    test_case.failure = ("Exception occured; type {0} message {1}".format(
      e.__class__, e.message))
  finally:
    test_case.time = time.time() - start
    if args.artifacts_path:
      test_util.create_junit_xml_file(
        [test_case],
        args.artifacts_path + "/junit_" + test_func.__name__ + ".xml")


def parse_runtime_params(args):
  salt = uuid.uuid4().hex[0:4]

  if "environment" in args and args.environment:
    env = args.environment
  else:
    env = "test-env-{0}".format(salt)

  name = None
  namespace = None
  for pair in args.params.split(","):
    k, v = pair.split("=", 1)
    if k == "name":
      name = v

    if k == "namespace":
      namespace = v

  if not name:
    raise ValueError("name must be provided as a parameter.")

  if not namespace:
    raise ValueError("namespace must be provided as a parameter.")

  return namespace, name, env


def add_common_args(parser):
  """Add a set of common parser arguments."""
  parser.add_argument(
    "--project", default=None, type=str, help=("The project to use."))

  parser.add_argument(
    "--cluster", default=None, type=str, help=("The name of the cluster."))

  parser.add_argument(
    "--app_dir",
    default=None,
    type=str,
    help="Directory containing the ksonnet app.")

  parser.add_argument(
    "--component",
    default=None,
    type=str,
    help="The ksonnet component of the job to run.")

  parser.add_argument(
    "--params",
    default=None,
    type=str,
    help="Comma separated list of key value pairs to set on the component.")

  parser.add_argument(
    "--zone",
    default="us-east1-d",
    type=str,
    help=("The zone for the cluster."))

  parser.add_argument(
    "--artifacts_path",
    default="",
    type=str,
    help="Where to write the test artifacts (e.g. junit xml file).")

  parser.add_argument(
    "--tfjob_version",
    default="v1",
    type=str,
    help="The TFJob version to use.")

  parser.add_argument(
    "--environment",
    default=None,
    type=str,
    help="(Optional) the name for the ksonnet environment; if not specified "
    "a random one is created.")

  parser.add_argument(
    "--num_trials",
    default=1,
    type=int,
    help="Number of times to run this test.")

  parser.add_argument(
    "--skip_tests",
    default=None,
    type=str,
    help="A comma delimited list of tests to skip.")


def main(module=None):  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO)  # pylint: disable=too-many-locals
  logging.basicConfig(
    level=logging.INFO,
    format=('%(levelname)s|%(asctime)s'
            '|%(pathname)s|%(lineno)d| %(message)s'),
    datefmt='%Y-%m-%dT%H:%M:%S',
  )

  # util.maybe_activate_service_account()

  parser = argparse.ArgumentParser(description="Run a TFJob test.")
  add_common_args(parser)

  args = parser.parse_args()
  test_module = import_module(module)
  skip_tests = []
  if args.skip_tests:
    skip_tests = args.skip_tests.split(",")

  types = dir(test_module)
  for t_name in types:
    t = getattr(test_module, t_name)
    if inspect.isclass(t) and issubclass(t, test_util.TestCase):
      logging.info("Loading test case: %s", t_name)
      test_case = t(args)
      funcs = dir(test_case)

      for f in funcs:
        if f.startswith("test_") and not f in skip_tests:
          test_func = getattr(test_case, f)
          logging.info("Invoking test method: %s", test_func)
          run_test(test_case, test_func, args)


if __name__ == "__main__":
  main()
