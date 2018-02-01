"""Test runner runs a TFJob test."""

import argparse
import logging
import os
import time
import uuid

import jinja2
import yaml

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

  t = test_util.TestCase()
  t.class_name = "tfjob_test"
  t.name = os.path.basename(args.spec)

  loader = jinja2.FileSystemLoader(os.path.dirname(args.spec))

  if not args.image_tag:
    raise ValueError("--image_tag must be provided.")

  logging.info("Loading spec from %s with image_tag=%s", args.spec, args.image_tag)
  spec_contents = jinja2.Environment(loader=loader).get_template(
    os.path.basename(args.spec)).render(image_tag=args.image_tag)

  spec = yaml.load(spec_contents)

  # Make the job name unique.
  spec["metadata"]["name"] += "-" + uuid.uuid4().hex[0:4]
  try:
    start = time.time()
    api_response = tf_job_client.create_tf_job(api_client, spec)
    namespace = api_response["metadata"]["namespace"]
    name = api_response["metadata"]["name"]

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
    # We want to catch all exceptions because we warm the test as failed.
    t.failure = e.message
  finally:
    t.time = time.time() - start
    if args.junit_path:
      test_util.create_junit_xml_file([t], args.junit_path, gcs_client)

def add_common_args(parser):
  """Add a set of common parser arguments."""

  parser.add_argument(
    "--spec",
    default=None,
    type=str,
    required=True,
    help="Path to the YAML file specifying the test to run.")

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
    "--image_tag",
    default=None,
    type=str,
    help="The tag for the docker image to use.")

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

  if os.getenv("GOOGLE_APPLICATION_CREDENTIALS"):
    logging.info("GOOGLE_APPLICATION_CREDENTIALS is set; configuring gcloud "
                 "to use service account.")
    # Since a service account is set tell gcloud to use it.
    util.run(["gcloud", "auth", "activate-service-account", "--key-file=" +
              os.getenv("GOOGLE_APPLICATION_CREDENTIALS")])

  parser = build_parser()

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)

if __name__ == "__main__":
  main()
