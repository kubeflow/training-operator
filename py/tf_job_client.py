"""Some utility functions for working with TFJobs."""

import datetime
import json
import logging
import time

from kubernetes import client as k8s_client
from kubernetes.client.rest import ApiException

from py import util

TF_JOB_GROUP = "kubeflow.org"
TF_JOB_VERSION = "v1alpha1"
TF_JOB_PLURAL = "tfjobs"
TF_JOB_KIND = "TFJob"

def create_tf_job(client, spec):
  """Create a TFJob.

  Args:
    client: A K8s api client.
    spec: The spec for the job.
  """
  crd_api = k8s_client.CustomObjectsApi(client)
  try:
    # Create a Resource
    namespace = spec["metadata"].get("namespace", "default")
    api_response = crd_api.create_namespaced_custom_object(
        TF_JOB_GROUP, TF_JOB_VERSION, namespace, TF_JOB_PLURAL, spec)
    logging.info("Created job %s", api_response["metadata"]["name"])
    return api_response
  except ApiException as e:
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
              "apis_fqdn_v1_namespaces_namespace_resource_post. body: %s"),
              e.body)
        raise
      message = body.get("message")

    logging.error(
        ("Exception when calling DefaultApi->"
         "apis_fqdn_v1_namespaces_namespace_resource_post: %s"),
          message)
    raise e

def log_status(tf_job):
  """A callback to use with wait_for_job."""
  logging.info("Job %s in namespace %s; phase=%s, state=%s,",
           tf_job["metadata"]["name"],
           tf_job["metadata"]["namespace"],
           tf_job["status"]["phase"],
           tf_job["status"]["state"])

def wait_for_job(client, namespace, name,
                 timeout=datetime.timedelta(minutes=5),
                 polling_interval=datetime.timedelta(seconds=30),
                 status_callback=None):
  """Wait for the specified job to finish.

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
    results = crd_api.get_namespaced_custom_object(
        TF_JOB_GROUP, TF_JOB_VERSION, namespace, TF_JOB_PLURAL, name)

    if status_callback:
      status_callback(results)

    if results["status"]["phase"] == "Done":
      return results

    if datetime.datetime.now() + polling_interval > end_time:
      raise util.TimeoutError(
        "Timeout waiting for job {0} in namespace {1} to finish.".format(
          name, namespace))

    time.sleep(polling_interval.seconds)

  # Linter complains if we don't have a return statement even though
  # this code is unreachable.
  return None
