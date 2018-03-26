"""Test runner runs a TFJob test."""

import argparse
import datetime
import httplib
import logging
import json
import os
import time
import uuid

from kubernetes import client as k8s_client
from kubernetes.client import rest

from google.cloud import storage  # pylint: disable=no-name-in-module
from py import test_util
from py import util
from py import tf_job_client


def wait_for_delete(client,
                    namespace,
                    name,
                    timeout=datetime.timedelta(minutes=5),
                    polling_interval=datetime.timedelta(seconds=30),
                    status_callback=None):
  """Wait for the specified job to be deleted.

  Args:
    client: K8s api client.
    namespace: namespace for the job.
    name: Name of the job.
    timeout: How long to wait for the job.
    polling_interval: How often to poll for the status of the job.
    status_callback: (Optional): Callable. If supplied this callable is
      invoked after we poll the job. Callable takes a single argument which
      is the job.
  """
  crd_api = k8s_client.CustomObjectsApi(client)
  end_time = datetime.datetime.now() + timeout
  while True:
    try:
      results = crd_api.get_namespaced_custom_object(
        tf_job_client.TF_JOB_GROUP, tf_job_client.TF_JOB_VERSION, namespace,
        tf_job_client.TF_JOB_PLURAL, name)
    except rest.ApiException as e:
      if e.status == httplib.NOT_FOUND:
        return
      raise
    if status_callback:
      status_callback(results)

    if datetime.datetime.now() + polling_interval > end_time:
      raise util.TimeoutError(
        "Timeout waiting for job {0} in namespace {1} to be deleted.".format(
          name, namespace))

    time.sleep(polling_interval.seconds)


def get_labels(name, runtime_id, replica_type=None, replica_index=None):
  """Return labels.
  """
  labels = {
    "kubeflow.org": "",
    "tf_job_name": name,
    "runtime_id": runtime_id,
  }
  if replica_type:
    labels["job_type"] = replica_type

  if replica_index:
    labels["task_index"] = replica_index
  return labels


def to_selector(labels):
  parts = []
  for k, v in labels.iteritems():
    parts.append("{0}={1}".format(k, v))

  return ",".join(parts)


def list_pods(client, namespace, label_selector):
  core = k8s_client.CoreV1Api(client)
  try:
    pods = core.list_namespaced_pod(namespace, label_selector=label_selector)
    return pods
  except rest.ApiException as e:
    message = ""
    if e.message:
      message = e.message
    if e.body:
      try:
        body = json.loads(e.body)
      except ValueError:
        # There was a problem parsing the body of the response as json.
        logging.error(
          ("Exception when calling DefaultApi->"
           "apis_fqdn_v1_namespaces_namespace_resource_post. body: %s"), e.body)
        raise
      message = body.get("message")

    logging.error(("Exception when calling DefaultApi->"
                   "apis_fqdn_v1_namespaces_namespace_resource_post: %s"),
                  message)
    raise e


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
    util.run(
      ["ks", "param", "set", "--env=" + env, args.component, k, v],
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
    # We repeat the test multiple times.
    # This ensures that if we delete the job we can create a new job with the
    # same name.

    # TODO(jlewi): We should make this an argument.
    num_trials = 2

    for trial in range(num_trials):
      logging.info("Trial %s", trial)
      util.run(["ks", "apply", env, "-c", args.component], cwd=args.app_dir)

      logging.info("Created job %s in namespaces %s", name, namespace)
      results = tf_job_client.wait_for_job(
        api_client, namespace, name, status_callback=tf_job_client.log_status)

      if results.get("status", {}).get("state", {}).lower() != "succeeded":
        t.failure = "Trial {0} Job {1} in namespace {2} in state {3}".format(
          trial, name, namespace,
          results.get("status", {}).get("state", None))
        logging.error(t.failure)
        break

      runtime_id = results.get("spec", {}).get("RuntimeId")
      logging.info("Trial %s Job %s in namespace %s runtime ID %s", trial, name,
                   namespace, runtime_id)

      # TODO(jlewi): We should check that pods were created for each replica
      pod_labels = get_labels(name, runtime_id)
      pod_selector = to_selector(pod_labels)
      pods = list_pods(api_client, namespace, pod_selector)

      logging.info("Trial %s selector: %s matched %s pods", trial, pod_selector,
                   len(pods.items))

      if not pods.items:
        t.failure = ("Trial {0} Job {1} in namespace {2} no pods found for "
                     " selector {3}").format(trial, name, namespace,
                                             pod_selector)
        logging.error(t.failure)
        break

      tf_job_client.delete_tf_job(api_client, namespace, name)

      logging.info("Waiting for job %s in namespaces %s to be deleted.", name, namespace)
      wait_for_delete(
        api_client, namespace, name, status_callback=tf_job_client.log_status)

      # Verify the pods have been deleted. tf_job_client uses foreground
      # deletion so there shouldn't be any resources for the job left
      # once the job is gone.
      pods = list_pods(api_client, namespace, pod_selector)

      logging.info("Trial %s selector: %s matched %s pods", trial, pod_selector,
                   len(pods.items))

      if pods.items:
        t.failure = ("Trial {0} Job {1} in namespace {2} pods found for "
                     " selector {3}; pods\n{4}").format(trial, name, namespace,
                                                        pod_selector, pods)
        logging.error(t.failure)
        break

      logging.info("Trial %s all pods deleted.", trial)

    # TODO(jlewi):
    #  Here are some validation checks to run:
    #  1. Check that all resources are garbage collected.
    # TODO(jlewi): Add an option to add chaos and randomly kill various resources?
    # TODO(jlewi): Are there other generic validation checks we should
    # run.
  except util.TimeoutError:
    t.failure = "Timeout waiting for {0} in namespace {1} to finish.".format(
      name, namespace)
    logging.error(t.failure)
  except Exception as e:  # pylint: disable-msg=broad-except
    # TODO(jlewi): I'm observing flakes where the exception has message "status"
    # in an effort to try to nail down this exception we print out more
    # information about the exception.
    logging.error("There was a problem running the job; Exception %s", e)
    logging.error("There was a problem running the job; Exception "
                  "message: %s", e.message)
    logging.error("Exception type: %s", e.__class__)
    logging.error("Exception args: %s", e.args)
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
    datefmt='%Y-%m-%dT%H:%M:%S',
  )

  util.maybe_activate_service_account()

  parser = build_parser()

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)


if __name__ == "__main__":
  main()
