"""Test runner runs a TFJob test."""

import argparse
import logging
import os
import time
import uuid

from kubernetes import client as k8s_client
from google.cloud import storage  # pylint: disable=no-name-in-module
from py import test_util
from py import util
from py import tf_job_client


def run_test(args):
  """Run a test."""
  gcs_client = storage.Client(project=args.project)
  project = args.project
  cluster_name = args.cluster
  zone = args.zone
  util.configure_kubectl(project, zone, cluster_name)
  util.load_kube_config()

  api_client = k8s_client.ApiClient()

  salt = uuid.uuid4().hex[0:4]

  # Create a new environment for this run
  env = "test-env-{0}".format(salt)

  util.run(["ks", "env", "add", env], cwd=args.app_dir)

  name = None
  namespace = None
  for pair in args.params.split(","):
    k, v = pair.split("=", 1)
    if k == "name":
      name = v

    if k == "namespace":
      namespace = v
    util.run(["ks", "param", "set", "--env=" + env, args.component, k, v],
             cwd=args.app_dir)

  if not name:
    raise ValueError("name must be provided as a parameter.")

  t = test_util.TestCase()
  t.class_name = "tfjob_test"
  t.name = os.path.basename(name)

  if not namespace:
    raise ValueError("namespace must be provided as a parameter.")

  start = time.time()

  try:
    util.run(["ks", "apply", env, "-c", args.component],
              cwd=args.app_dir)

    logging.info("Created job %s in namespaces %s", name, namespace)
    results = tf_job_client.wait_for_job(api_client, namespace, name,
                                         status_callback=tf_job_client.log_status)

    if results["status"]["state"].lower() != "succeeded":
      t.failure = "Job {0} in namespace {1} in state {2}".format(
        name, namespace, results["status"]["state"])

    # TODO(jlewi):
    #  Here are some validation checks to run:
    #  1. Check tensorboard is created if its part of the job spec.
    #  2. Check that all resources are garbage collected.
    # TODO(jlewi): Add an option to add chaos and randomly kill various resources?
    # TODO(jlewi): Are there other generic validation checks we should
    # run.
  except util.TimeoutError:
    t.failure = "Timeout waiting for {0} in namespace {1} to finish.".format(
        name, namespace)
  except Exception as e: # pylint: disable-msg=broad-except
    logging.error("There was a problem running the job; Exception %s; "
                  "Exception message: %s",
                  "Exception args: %s",
                  "Exception type: %s", e, e.message, e.args, e.__class__)
    # We want to catch all exceptions because we want the test as failed.
    t.failure = e.message
  finally:
    t.time = time.time() - start
    if args.junit_path:
      test_util.create_junit_xml_file([t], args.junit_path, gcs_client)

def add_common_args(parser):
  """Add a set of common parser arguments."""

  parser.add_argument(
    "--project",
    default=None,
    type=str,
    help=("The project to use."))

  parser.add_argument(
    "--cluster",
    default=None,
    type=str,
    help=("The name of the cluster."))

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

def build_parser():
  # create the top-level parser
  parser = argparse.ArgumentParser(
    description="Run a TFJob test.")
  subparsers = parser.add_subparsers()

  parser_test = subparsers.add_parser(
    "test",
    help="Run a tfjob test.")

  add_common_args(parser_test)
  parser_test.set_defaults(func=run_test)

  return parser

def main():  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO) # pylint: disable=too-many-locals
  logging.basicConfig(level=logging.INFO,
                      format=('%(levelname)s|%(asctime)s'
                              '|%(pathname)s|%(lineno)d| %(message)s'),
                      datefmt='%Y-%m-%dT%H:%M:%S',
                      )

  util.maybe_activate_service_account()

  parser = build_parser()

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)

if __name__ == "__main__":
  main()
