"""Test runner runs a TFJob test."""

import argparse
import inspect
import json
import logging
import retrying
import time

from importlib import import_module

from google.cloud import storage  # pylint: disable=no-name-in-module
from kubeflow.testing import test_util, util
from py import util as tf_operator_util
#from types import ClassType


# One of the reasons we set so many retries and a random amount of wait
# between retries is because we have multiple tests running in parallel
# that are all modifying the same ksonnet app via ks. I think this can
# lead to failures.
@retrying.retry(stop_max_attempt_number=10, wait_random_min=1000,
                wait_random_max=10000)
def run_test(test_case, test_func, args):  # pylint: disable=too-many-branches,too-many-statements
  """Run a test."""
  gcs_client = storage.Client(project=args.project)
  project = args.project
  cluster_name = args.cluster
  zone = args.zone
  # TODO(jlewi): When using GKE we should copy the .kube config and any other
  # files to the test directory. We should then set the environment variable
  # KUBECONFIG to point at that file. This should prevent us from having
  # to rerun util.configure_kubectl on each step. Instead we could run it once
  # as part of GKE cluster creation and store the config in the NFS directory.
  # This would make the handling of credentials
  # and KUBECONFIG more consistent between GKE and minikube and eventually
  # this could be extended to other K8s deployments.
  if cluster_name:
    util.configure_kubectl(project, zone, cluster_name)
  util.load_kube_config()

  #t = test_helper.TestCase(test_func)
  #t.class_name = "tfjob_test"
  #t.name = os.path.basename(args.test_method)

  start = time.time()

  #module = import_module("py." + args.test_module)
  #test_func = getattr(module, args.test_method)

  try: # pylint: disable=too-many-nested-blocks
    # We repeat the test multiple times.
    # This ensures that if we delete the job we can create a new job with the
    # same name.

    # TODO(jlewi): We should make this an argument.
    num_trials = 2
    logging.info("tfjob_version=%s", args.tfjob_version)

    for trial in range(num_trials):
      logging.info("Trial %s", trial)
      test_func()
      #test_result = test_func(t, args)
      #if not test_result:
      #  break

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
    if args.junit_path:
      test_util.create_junit_xml_file([test_case], args.junit_path, gcs_client)


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
    "--junit_path",
    default="",
    type=str,
    help="Where to write the junit xml file with the results.")

  parser.add_argument(
    "--tfjob_version",
    default="v1alpha1",
    type=str,
    help="The TFJob version to use.")

  parser.add_argument(
    "--environment",
    default=None,
    type=str,
    help="(Optional) the name for the ksonnet environment; if not specified "
         "a random one is created.")

def main(module=None):  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO)  # pylint: disable=too-many-locals
  logging.basicConfig(
    level=logging.INFO,
    format=('%(levelname)s|%(asctime)s'
            '|%(pathname)s|%(lineno)d| %(message)s'),
    datefmt='%Y-%m-%dT%H:%M:%S',)

  util.maybe_activate_service_account()

  parser = argparse.ArgumentParser(description="Run a TFJob test.")
  add_common_args(parser)

  args = parser.parse_args()
  test_module = import_module(module)
  #for x, y in test_module.__dict__.items():
  #  logging.info(">>>> x %s", x)
  #  logging.info(">>>> y %s", y)

  types = dir(test_module):
  for t_name in types:
    logging.info(">>>> t_name: %s", t_name)
    t = getattr(test_module, t_name)
    if inspect.isclass(t) and issubclass(t, test_util.TestCase): 
      #type(y) is test_util.TestCase: #ClassType and issubclass(y, test_util.TestCase()):
      test_case = t()
      funcs = dir(test_case)

      for f in funcs:
        logging.info(">>>> func: %s", f)
        if f.startswith("test_"):
          test_func = getattr(test_case, f)
          logging.info(">>>> tf: %s", test_func)
          #run_test(tf, args)
          run_test(test_case, test_func, args)


if __name__ == "__main__":
  main()
