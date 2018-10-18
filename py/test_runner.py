"""Test runner runs a TFJob test."""

import argparse
import logging
import json
import os
import retrying
import time
import yaml

from importlib import import_module

from google.cloud import storage  # pylint: disable=no-name-in-module
from kubeflow.testing import util
from py import test_util
from py import util as tf_operator_util


def get_runconfig(master_host, namespace, target):
  """Issue a request to get the runconfig of the specified replica running test_server.

    Args:
    master_host: The IP address of the master e.g. https://35.188.37.10
    namespace: The namespace
    target: The K8s service corresponding to the pod to call.
  """
  response = tf_operator_util.send_request(master_host, namespace, target, "runconfig", {})
  return yaml.load(response)


def verify_runconfig(master_host, namespace, job_name, replica, num_ps, num_workers):
  """Verifies that the TF RunConfig on the specified replica is the same as expected.

    Args:
    master_host: The IP address of the master e.g. https://35.188.37.10
    namespace: The namespace
    job_name: The name of the TF job
    replica: The replica type (chief, ps, or worker)
    num_ps: The number of PS replicas
    num_workers: The number of worker replicas
  """
  is_chief = True
  num_replicas = 1
  if replica == "ps":
    is_chief = False
    num_replicas = num_ps
  elif replica == "worker":
    is_chief = False
    num_replicas = num_workers

  # Construct the expected cluster spec
  chief_list = ["{name}-chief-0:2222".format(name=job_name)]
  ps_list = []
  for i in range(num_ps):
    ps_list.append("{name}-ps-{index}:2222".format(name=job_name, index=i))
  worker_list = []
  for i in range(num_workers):
    worker_list.append("{name}-worker-{index}:2222".format(name=job_name, index=i))
  cluster_spec = {
    "chief": chief_list,
    "ps": ps_list,
    "worker": worker_list,
  }

  for i in range(num_replicas):
    full_target = "{name}-{replica}-{index}".format(name=job_name, replica=replica.lower(), index=i)
    actual_config = get_runconfig(master_host, namespace, full_target)
    expected_config = {
      "task_type": replica,
      "task_id": i,
      "cluster_spec": cluster_spec,
      "is_chief": is_chief,
      "master": "grpc://{target}:2222".format(target=full_target),
      "num_worker_replicas": num_workers + 1, # Chief is also a worker
      "num_ps_replicas": num_ps,
    }
    # Compare expected and actual configs
    if actual_config != expected_config:
      msg = "Actual runconfig differs from expected. Expected: {0} Actual: {1}".format(
        str(expected_config), str(actual_config))
      logging.error(msg)
      raise RuntimeError(msg)


# One of the reasons we set so many retries and a random amount of wait
# between retries is because we have multiple tests running in parallel
# that are all modifying the same ksonnet app via ks. I think this can
# lead to failures.
@retrying.retry(stop_max_attempt_number=10, wait_random_min=1000,
                wait_random_max=10000)
def run_test(args):  # pylint: disable=too-many-branches,too-many-statements
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

  #api_client = k8s_client.ApiClient()
  #masterHost = api_client.configuration.host

  t = test_util.TestCase()
  t.class_name = "tfjob_test"
  #namespace, name, _ = ks_util.setup_ks_app(args)
  t.name = os.path.basename(args.test_method)

  start = time.time()

  module = import_module("py." + args.test_module)
  test_func = getattr(module, args.test_method)

  try: # pylint: disable=too-many-nested-blocks
    # We repeat the test multiple times.
    # This ensures that if we delete the job we can create a new job with the
    # same name.

    # TODO(jlewi): We should make this an argument.
    num_trials = 2
    logging.info("tfjob_version=%s", args.tfjob_version)

    for trial in range(num_trials):
      logging.info("Trial %s", trial)
      test_result = test_func(t, args)
      if not test_result:
        break

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
    t.failure = "Timeout waiting for job to finish: " + spec
    logging.exception(t.failure)
  except Exception as e:  # pylint: disable-msg=broad-except
    # TODO(jlewi): I'm observing flakes where the exception has message "status"
    # in an effort to try to nail down this exception we print out more
    # information about the exception.
    logging.exception("There was a problem running the job; Exception %s", e)
    # We want to catch all exceptions because we want the test as failed.
    t.failure = ("Exception occured; type {0} message {1}".format(
      e.__class__, e.message))
  finally:
    t.time = time.time() - start
    if args.junit_path:
      test_util.create_junit_xml_file([t], args.junit_path, gcs_client)


def add_common_args(parser):
  """Add a set of common parser arguments."""

  parser.add_argument(
    "--test_module", default=None, type=str, help=("The test module to import."))

  parser.add_argument(
    "--test_method", default=None, type=str, help=("The test method to invoke."))

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
    "--shutdown_policy",
    default=None,
    type=str,
    help="The shutdown policy. This must be set if we need to issue "
         "an http request to the test-app server to exit before the job will "
         "finish.")

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

  parser.add_argument(
    "--verify_clean_pod_policy",
    default=None,
    type=str,
    help="(Optional) the clean pod policy (None, Running, or All).")

  parser.add_argument(
    "--verify_runconfig",
    dest="verify_runconfig",
    action="store_true",
    help="(Optional) verify runconfig in each replica.")


def build_parser():
  # create the top-level parser
  parser = argparse.ArgumentParser(description="Run a TFJob test.")
  subparsers = parser.add_subparsers()

  parser_test = subparsers.add_parser("test", help="Run a tfjob test.")

  add_common_args(parser_test)
  parser_test.set_defaults(func=run_test)

  return parser


def main():  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO)  # pylint: disable=too-many-locals
  logging.basicConfig(
    level=logging.INFO,
    format=('%(levelname)s|%(asctime)s'
            '|%(pathname)s|%(lineno)d| %(message)s'),
    datefmt='%Y-%m-%dT%H:%M:%S',)

  util.maybe_activate_service_account()

  parser = build_parser()

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)


if __name__ == "__main__":
  main()
