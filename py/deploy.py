#!/usr/bin/python
"""Deploy/manage K8s clusters and the operator.

This binary is primarily intended for use in managing resources for our tests.
"""

import argparse
import datetime
import logging
import subprocess
import time
import uuid

from kubernetes import client as k8s_client
from kubernetes.client import rest
from googleapiclient import discovery
from google.cloud import storage  # pylint: disable=no-name-in-module

from py import test_util
from py import util


def _setup_namespace(api_client, name):
  """Create the namespace for the test.
  """

  api = k8s_client.CoreV1Api(api_client)
  namespace = k8s_client.V1Namespace()
  namespace.api_version = "v1"
  namespace.kind = "Namespace"
  namespace.metadata = k8s_client.V1ObjectMeta(
    name=name, labels={
      "app": "tf-job-test",
    })

  try:
    logging.info("Creating namespace %s", namespace.metadata.name)
    namespace = api.create_namespace(namespace)
    logging.info("Namespace %s created.", namespace.metadata.name)
  except rest.ApiException as e:
    if e.status == 409:
      logging.info("Namespace %s already exists.", namespace.metadata.name)
    else:
      raise


# TODO(jlewi): We should probably make this a reusable function since a
# lot of test code code use it.
def ks_deploy(app_dir, component, params, env=None, account=None):
  """Deploy the specified ksonnet component.

  Args:
    app_dir: The ksonnet directory
    component: Name of the component to deployed
    params: A dictionary of parameters to set; can be empty but should not be
      None.
    env: (Optional) The environment to use, if none is specified a new one
      is created.
    account: (Optional) The account to use.

  Raises:
    ValueError: If input arguments aren't valid.
  """
  if not component:
    raise ValueError("component can't be None.")

  # TODO(jlewi): It might be better if the test creates the app and uses
  # the latest stable release of the ksonnet configs. That however will cause
  # problems when we make changes to the TFJob operator that require changes
  # to the ksonnet configs. One advantage of checking in the app is that
  # we can modify the files in vendor if needed so that changes to the code
  # and config can be submitted in the same pr.
  now = datetime.datetime.now()
  if not env:
    env = "e2e-" + now.strftime("%m%d-%H%M-") + uuid.uuid4().hex[0:4]

  logging.info("Using app directory: %s", app_dir)

  util.run(["ks", "env", "add", env], cwd=app_dir)

  for k, v in params.iteritems():
    util.run(
      ["ks", "param", "set", "--env=" + env, component, k, v], cwd=app_dir)

  apply_command = ["ks", "apply", env, "-c", component]
  if account:
    apply_command.append("--as=" + account)
  util.run(apply_command, cwd=app_dir)


def setup(args):
  """Setup a GKE cluster for TensorFlow jobs.

  Args:
    args: Command line arguments that control the setup process.
  """
  gke = discovery.build("container", "v1")

  project = args.project
  cluster_name = args.cluster
  zone = args.zone
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
    }
  }

  if args.accelerators:
    # TODO(jlewi): Stop enabling Alpha once GPUs make it out of Alpha
    cluster_request["cluster"]["enableKubernetesAlpha"] = True

    cluster_request["cluster"]["nodeConfig"]["accelerators"] = []
    for accelerator_spec in args.accelerators:
      accelerator_type, accelerator_count = accelerator_spec.split("=", 1)
      cluster_request["cluster"]["nodeConfig"]["accelerators"].append({
        "acceleratorCount":
        accelerator_count,
        "acceleratorType":
        accelerator_type,
      })

  util.create_cluster(gke, project, zone, cluster_request)

  util.configure_kubectl(project, zone, cluster_name)

  util.load_kube_config()
  # Create an API client object to talk to the K8s master.
  api_client = k8s_client.ApiClient()

  t = test_util.TestCase()
  try:
    start = time.time()

    params = {
      "tfJobImage": args.image,
      "name": "kubeflow-core",
      "namespace": args.namespace,
    }

    component = "core"

    account = util.run_and_output(
      ["gcloud", "config", "get-value", "account", "--quiet"]).strip()
    logging.info("Using GCP account %s", account)
    util.run([
      "kubectl", "create", "clusterrolebinding", "default-admin",
      "--clusterrole=cluster-admin", "--user=" + account
    ])

    _setup_namespace(api_client, args.namespace)
    ks_deploy(args.test_app_dir, component, params, account=account)

    # Setup GPUs.
    util.setup_cluster(api_client)

    # Verify that the TfJob operator is actually deployed.
    tf_job_deployment_name = "tf-job-operator"
    logging.info("Verifying TfJob controller started.")

    # TODO(jlewi): We should verify the image of the operator is the correct.
    util.wait_for_deployment(api_client, args.namespace, tf_job_deployment_name)

  # Reraise the exception so that the step fails because there's no point
  # continuing the test.
  except subprocess.CalledProcessError as e:
    t.failure = "kubeflow-deploy failed;\n" + (e.output or "")
    raise
  except util.TimeoutError as e:
    t.failure = e.message
    raise
  finally:
    t.time = time.time() - start
    t.name = "kubeflow-deploy"
    t.class_name = "GKE"
    gcs_client = storage.Client(project=args.project)
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
    "--project", default=None, type=str, help=("The project to use."))
  parser.add_argument(
    "--cluster", default=None, type=str, help=("The name of the cluster."))
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
  logging.getLogger().setLevel(logging.INFO)  # pylint: disable=too-many-locals

  util.maybe_activate_service_account()

  # create the top-level parser
  parser = argparse.ArgumentParser(description="Setup clusters for testing.")
  subparsers = parser.add_subparsers()

  #############################################################################
  # setup
  #
  parser_setup = subparsers.add_parser(
    "setup", help="Setup a cluster for testing.")

  parser_setup.add_argument(
    "--accelerator",
    dest="accelerators",
    action="append",
    help="Accelerator to add to the cluster. Should be of the form type=count.")

  parser_setup.set_defaults(func=setup)
  add_common_args(parser_setup)

  parser_setup.add_argument(
    "--test_app_dir",
    help="The directory containing the ksonnet app used for testing.",
  )

  now = datetime.datetime.now()
  parser_setup.add_argument(
    "--namespace",
    default="kubeflow-" + now.strftime("%m%d-%H%M-") + uuid.uuid4().hex[0:4],
    help="The directory containing the ksonnet app used for testing.",
  )

  parser_setup.add_argument(
    "--image",
    help="The image to use",
  )

  #############################################################################
  # teardown
  #
  parser_teardown = subparsers.add_parser(
    "teardown", help="Teardown the cluster.")
  parser_teardown.set_defaults(func=teardown)
  add_common_args(parser_teardown)

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)


if __name__ == "__main__":
  main()
