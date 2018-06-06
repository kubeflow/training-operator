"""Some utility functions for working with TFJobs."""

import datetime
import json
import logging
import multiprocessing
import time

from kubernetes import client as k8s_client
from kubernetes.client.rest import ApiException

from py import util

TF_JOB_GROUP = "kubeflow.org"
TF_JOB_PLURAL = "tfjobs"
TF_JOB_KIND = "TFJob"

# How long to wait in seconds for requests to the ApiServer
TIMEOUT = 120

def create_tf_job(client, spec, version="v1alpha1"):
  """Create a TFJob.

  Args:
    client: A K8s api client.
    spec: The spec for the job.
  """
  crd_api = k8s_client.CustomObjectsApi(client)
  try:
    # Create a Resource
    namespace = spec["metadata"].get("namespace", "default")
    thread = crd_api.create_namespaced_custom_object(
      TF_JOB_GROUP, version, namespace, TF_JOB_PLURAL, spec, async=True)
    api_response = thread.get(TIMEOUT)
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
           "apis_fqdn_v1_namespaces_namespace_resource_post. body: %s"), e.body)
        raise
      message = body.get("message")

    logging.error(("Exception when calling DefaultApi->"
                   "apis_fqdn_v1_namespaces_namespace_resource_post: %s"),
                  message)
    raise e


def delete_tf_job(client, namespace, name, version="v1alpha1"):
  crd_api = k8s_client.CustomObjectsApi(client)
  try:
    body = {
      # Set garbage collection so that job won't be deleted until all
      # owned references are deleted.
      "propagationPolicy": "Foreground",
    }
    logging.info("Deleting job %s.%s", namespace, name)
    thread = crd_api.delete_namespaced_custom_object(
      TF_JOB_GROUP, version, namespace, TF_JOB_PLURAL, name, body,
      async=True)
    api_response = thread.get(TIMEOUT)
    logging.info("Deleting job %s.%s returned: %s", namespace, name, api_response)
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
           "apis_fqdn_v1_namespaces_namespace_resource_post. body: %s"), e.body)
        raise
      message = body.get("message")

    logging.error(("Exception when calling DefaultApi->"
                   "apis_fqdn_v1_namespaces_namespace_resource_post: %s"),
                  message)
    raise e


def log_status(tf_job):
  """A callback to use with wait_for_job."""
  logging.info("Job %s in namespace %s; uid=%s; phase=%s, state=%s,",
               tf_job.get("metadata", {}).get("name"),
               tf_job.get("metadata", {}).get("namespace"),
               tf_job.get("metadata", {}).get("uid"),
               tf_job.get("status", {}).get("phase"),
               tf_job.get("status", {}).get("state"))


def wait_for_job(client,
                 namespace,
                 name,
                 version="v1alpha1",
                 timeout=datetime.timedelta(minutes=10),
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
    # By setting async=True ApiClient returns multiprocessing.pool.AsyncResult
    # If we don't set async=True then it could potentially block forever.
    thread = crd_api.get_namespaced_custom_object(
      TF_JOB_GROUP, version, namespace, TF_JOB_PLURAL, name, async=True)

    # Try to get the result but timeout.
    results = None
    try:
      results = thread.get(TIMEOUT)
    except multiprocessing.TimeoutError:
      logging.error("Timeout trying to get TFJob.")

    if results:
      if status_callback:
        status_callback(results)

      # If we poll the CRD quick enough status won't have been set yet.
      if version == "v1alpha1":
        if results.get("status", {}).get("phase", {}) == "Done":
          return results
      else:
        # For v1alpha2 check for non-empty completionTime
        if results.get("status", {}).get("completionTime", ""):
          return results


    if datetime.datetime.now() + polling_interval > end_time:
      raise util.TimeoutError(
        "Timeout waiting for job {0} in namespace {1} to finish.".format(
          name, namespace))

    time.sleep(polling_interval.seconds)

  # Linter complains if we don't have a return statement even though
  # this code is unreachable.
  return None
