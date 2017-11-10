#!/usr/bin/python
"""Deploy/manage K8s clusters and the operator.

This binary is primarily intended for use in managing resources for our tests.
"""


import argparse
import datetime
import glob
import json
import logging
import os
import shutil
import tempfile

import kubernetes
from kubernetes import client as k8s_client
from kubernetes import config as k8s_config

from googleapiclient import discovery, errors
from google.cloud import storage  # pylint: disable=no-name-in-module

from py import util

def setup(args):
  gke = discovery.build("container", "v1")

  project = args.project
  cluster_name = args.cluster
  zone = args.zone
  chart = args.chart
  machine_type = "n1-standard-8"

  # TODO(jlewi): Should make these command line arguments.
  use_gpu = False
  if use_gpu:
    accelerator = "nvidia-tesla-k80"
    accelerator_count = 1
  else:
    accelerator = None
    accelerator_count = 0

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
          "initialClusterVersion": "1.8.1-gke.1",
      }
  }

  if bool(accelerator) != (accelerator_count > 0):
      raise ValueError("If accelerator is set accelerator_count must be  > 0")

  if accelerator:
    # TODO(jlewi): Stop enabling Alpha once GPUs make it out of Alpha
    cluster_request["cluster"]["enableKubernetesAlpha"] = True

    cluster_request["cluster"]["nodeConfig"]["accelerators"] = [
        {
          "acceleratorCount": accelerator_count,
          "acceleratorType": accelerator,
        },
    ]

  util.create_cluster(gke, project, zone, cluster_request)

  util.configure_kubectl(project, zone, cluster_name)

  k8s_config.load_kube_config()

  # Create an API client object to talk to the K8s master.
  api_client = k8s_client.ApiClient()

  util.setup_cluster(api_client)

  if chart.startswith("gs://"):
    remote = chart
    chart = os.path.join(tempfile.gettempdir(), os.path.basename(chart))
    gcs_client = storage.Client(project=project)
    bucket_name, path = util.split_gcs_uri(remote)

    bucket = gcs_client.get_bucket(bucket_name)
    blob = bucket.blob(path)
    logging.info("Downloading %s to %", remote, chart)
    blob.download_to_filename(chart)

  util.run(["helm", "install", chart, "-n", "tf-job", "--wait", "--replace",
            "--set", "rbac.install=true,cloud=gke"])

def test(args):
  """Run the tests."""
  project = args.project
  cluster_name = args.cluster
  zone = args.zone
  util.configure_kubectl(project, zone, cluster_name)
  util.run(["helm", "test", "tf-job"])

def teardown(args):
  """Teardown the resources."""
  gke = discovery.build("container", "v1")

  project = args.project
  cluster_name = args.cluster
  zone = args.zone
  util.delete_cluster(gke, name, project, zone)

def add_common_args(parser):
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
  parser_teardown.set_defaults(func=setup)
  add_common_args(parser_teardown)

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)

if __name__ == "__main__":
  main()
