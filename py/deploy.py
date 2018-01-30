#!/usr/bin/python
"""Deploy/manage K8s clusters and the operator.

This binary is primarily intended for use in managing resources for our tests.
"""

import argparse
import logging
import os
import subprocess
import tempfile
import time

from kubernetes import client as k8s_client

from googleapiclient import discovery
from google.cloud import storage  # pylint: disable=no-name-in-module

from py import test_util
from py import util

def setup(args):
  """Setup a GKE cluster for TensorFlow jobs.

  Args:
    args: Command line arguments that control the setup process.
  """
  gke = discovery.build("container", "v1")

  project = args.project
  cluster_name = args.cluster
  zone = args.zone
  chart = args.chart
  machine_type = "n1-standard-8"

  cluster_request = {
    "cluster": {
        "name": cluster_name,
          "description": "A GKE cluster for TF.",
          "initialNodeCount": 1,
          "nodeConfig": {
            "machineType": machine_type,
              "oauthScopes": [
                "https://www.googleapis.com/auth/cloud-platform",
                ],
              },
          # TODO(jlewi): Stop pinning GKE version once 1.8 becomes the default.
          "initialClusterVersion": "1.8.5-gke.0",
      }
  }

  if args.accelerators:
    # TODO(jlewi): Stop enabling Alpha once GPUs make it out of Alpha
    cluster_request["cluster"]["enableKubernetesAlpha"] = True

    cluster_request["cluster"]["nodeConfig"]["accelerators"] = []
    for accelerator_spec in args.accelerators:
      accelerator_type, accelerator_count = accelerator_spec.split("=", 1)
      cluster_request["cluster"]["nodeConfig"]["accelerators"].append(
        {"acceleratorCount": accelerator_count,
         "acceleratorType": accelerator_type, })

  util.create_cluster(gke, project, zone, cluster_request)

  util.configure_kubectl(project, zone, cluster_name)

  util.load_kube_config()
  # Create an API client object to talk to the K8s master.
  api_client = k8s_client.ApiClient()

  util.setup_cluster(api_client)

  # A None gcs_client should be passed to test_util.create_junit_xml_file
  # unless chart.startswith("gs://"), e.g. https://storage.googleapis.com/...
  gcs_client = None

  if chart.startswith("gs://"):
    remote = chart
    chart = os.path.join(tempfile.gettempdir(), os.path.basename(chart))
    gcs_client = storage.Client(project=project)
    bucket_name, path = util.split_gcs_uri(remote)

    bucket = gcs_client.get_bucket(bucket_name)
    blob = bucket.blob(path)
    logging.info("Downloading %s to %s", remote, chart)
    blob.download_to_filename(chart)

  t = test_util.TestCase()
  try:
    start = time.time()
    util.run(["helm", "install", chart, "-n", "tf-job", "--wait", "--replace",
              "--set", "rbac.install=true,cloud=gke"])
    util.wait_for_deployment(api_client, "default", "tf-job-operator")
  except subprocess.CalledProcessError as e:
    t.failure = "helm install failed;\n" + (e.output or "")
  except util.TimeoutError as e:
    t.failure = e.message
  finally:
    t.time = time.time() - start
    t.name = "helm-tfjob-install"
    t.class_name = "GKE"
    test_util.create_junit_xml_file([t], args.junit_path, gcs_client)

def test(args):
  """Run the tests."""
  gcs_client = storage.Client(project=args.project)
  project = args.project
  cluster_name = args.cluster
  zone = args.zone
  util.configure_kubectl(project, zone, cluster_name)

  t = test_util.TestCase()
  try:
    start = time.time()
    util.run(["helm", "test", "tf-job"])
  except subprocess.CalledProcessError as e:
    t.failure = "helm test failed;\n" + (e.output or "")
    # Reraise the exception so that the prow job will fail and the test
    # is marked as a failure.
    # TODO(jlewi): It would be better to this wholistically; e.g. by
    # processing all the junit xml files and checking for any failures. This
    # should be more tractable when we migrate off Airflow to Argo.
    raise
  finally:
    t.time = time.time() - start
    t.name = "e2e-test"
    t.class_name = "GKE"
    test_util.create_junit_xml_file([t], args.junit_path, gcs_client)

def teardown(args):
  """Teardown the resources."""
  gke = discovery.build("container", "v1")

  project = args.project
  cluster_name = args.cluster
  zone = args.zone
  util.delete_cluster(gke, cluster_name, project, zone)

def add_common_args(parser):
  """Add common command line arguments to a parser.

  Args:
    parser: The parser to add command line arguments to.
  """
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
    "--zone",
    default="us-east1-d",
    type=str,
    help=("The zone for the cluster."))

  parser.add_argument(
    "--junit_path",
    default="",
    type=str,
    help="Where to write the junit xml file with the results.")

def main():  # pylint: disable=too-many-locals
  logging.getLogger().setLevel(logging.INFO) # pylint: disable=too-many-locals
  # create the top-level parser
  parser = argparse.ArgumentParser(
    description="Setup clusters for testing.")
  subparsers = parser.add_subparsers()

  #############################################################################
  # setup
  #
  parser_setup = subparsers.add_parser(
    "setup",
      help="Setup a cluster for testing.")

  parser_setup.add_argument(
    "--accelerator",
    dest="accelerators",
    action="append",
    help="Accelerator to add to the cluster. Should be of the form type=count.")

  parser_setup.set_defaults(func=setup)
  add_common_args(parser_setup)

  parser_setup.add_argument(
    "--chart",
    type=str,
    required=True,
    help="The path for the helm chart.")

  #############################################################################
  # test
  #
  parser_test = subparsers.add_parser(
    "test",
    help="Run the tests.")

  parser_test.set_defaults(func=test)
  add_common_args(parser_test)

  #############################################################################
  # teardown
  #
  parser_teardown = subparsers.add_parser(
    "teardown",
    help="Teardown the cluster.")
  parser_teardown.set_defaults(func=teardown)
  add_common_args(parser_teardown)

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)

if __name__ == "__main__":
  main()
