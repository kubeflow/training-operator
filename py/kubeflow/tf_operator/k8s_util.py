"""K8s util class for E2E tests."""

import datetime
import json
import logging
import re
import time

from kubeflow.testing import util
from kubernetes import client as k8s_client
from kubernetes.client import rest


def get_container_start_time(client, namespace, pod_selector, index, phase):
  """ get start time of container in the pod with pod_name,
  we assume there is only one container.

  Args:
    client: K8s api client.
    namespace: Namespace.
    pod_selector: Selector for the pods.
    index: Index of the pods
    phase: expected of the phase when getting the start time
  Returns:
    container_start_time: container start time in datetime datatype
  """
  pods = list_pods(client, namespace, pod_selector)
  logging.info("%s pods matched %s pods", len(pods.items), pod_selector)
  pod = pods.items[index]

  if phase == "Running":
    container_start_time = pod.status.container_statuses[
      0].state.running.started_at
  else:
    container_start_time = pod.status.container_statuses[
      0].state.terminated.started_at

  return container_start_time


def log_pods(pods):
  """Log information about pods."""
  for p in pods.items:
    logging.info("Pod name=%s Phase=%s", p.metadata.name, p.status.phase)


def wait_for_pods_to_be_in_phases(
    client,
    namespace,
    pod_selector,
    phases,
    timeout=datetime.timedelta(minutes=15),
    polling_interval=datetime.timedelta(seconds=30)):
  """Wait for the pods matching the selector to be in the specified state

  Args:
    client: K8s api client.
    namespace: Namespace.
    pod_selector: Selector for the pods.
    phases: List of desired phases
    timeout: How long to wait for the job.
    polling_interval: How often to poll for the status of the job.
    status_callback: (Optional): Callable. If supplied this callable is
      invoked after we poll the job. Callable takes a single argument which
      is the job.
  """
  time.sleep(polling_interval.seconds)
  end_time = datetime.datetime.now() + timeout
  while True:

    pods = list_pods(client, namespace, pod_selector)

    logging.info("%s pods matched %s pods", len(pods.items), pod_selector)

    is_match = True

    for p in pods.items:
      if p.status.phase not in phases:
        # for debug
        logging.info("pod in phase %s", p.status.phase)
        is_match = False

    if is_match and pods.items:
      logging.info("All pods in phase %s", phases)
      log_pods(pods)
      return pods

    if datetime.datetime.now() + polling_interval > end_time:
      logging.info("Latest pod phases")
      log_pods(pods)
      logging.error("Timeout waiting for pods to be in phase: %s", phases)
      raise util.TimeoutError(
        "Timeout waiting for pods to be in states %s" % phases)

    time.sleep(polling_interval.seconds)

  return None


def wait_for_pods_to_be_deleted(
    client,
    namespace,
    pod_selector,
    timeout=datetime.timedelta(minutes=10),
    polling_interval=datetime.timedelta(seconds=30)):
  """Wait for the specified job to be deleted.

  Args:
    client: K8s api client.
    namespace: Namespace.
    pod_selector: Selector for the pods.
    timeout: How long to wait for the job.
    polling_interval: How often to poll for the status of the job.
    status_callback: (Optional): Callable. If supplied this callable is
      invoked after we poll the job. Callable takes a single argument which
      is the job.
  """
  end_time = datetime.datetime.now() + timeout
  while True:
    pods = list_pods(client, namespace, pod_selector)

    logging.info("%s pods matched %s pods", len(pods.items), pod_selector)

    if not pods.items:
      return

    if datetime.datetime.now() + polling_interval > end_time:
      raise util.TimeoutError("Timeout waiting for pods to be deleted.")

    time.sleep(polling_interval.seconds)


def list_pods(client, namespace, label_selector):
  core = k8s_client.CoreV1Api(client)
  try:
    pods = core.list_namespaced_pod(namespace, label_selector=label_selector)
    return pods
  except rest.ApiException as e:
    message = ""
    if hasattr(e, "message"):
      message = e.message
    if hasattr(e, "body"):
      try:
        body = json.loads(e.body)
      except ValueError:
        # There was a problem parsing the body of the response as json.
        logging.exception(
          ("Exception when calling DefaultApi->"
           "apis_fqdn_v1_namespaces_namespace_resource_post. body: %s"), e.body)
        raise
      message = body.get("message")

    logging.exception(("Exception when calling DefaultApi->"
                       "apis_fqdn_v1_namespaces_namespace_resource_post: %s"),
                      message)
    raise e


def get_events(client, namespace, uid):
  """Get the events for the provided object."""
  core = k8s_client.CoreV1Api(client)
  try:
    # We can't filter by labels because events don't appear to have anyone
    # and I didn't see an easy way to get them.
    events = core.list_namespaced_event(namespace, limit=500)
  except rest.ApiException as e:
    message = ""
    if e.message:
      message = e.message
    if e.body:
      try:
        body = json.loads(e.body)
      except ValueError:
        # There was a problem parsing the body of the response as json.
        logging.exception(
          ("Exception when calling DefaultApi->"
           "apis_fqdn_v1_namespaces_namespace_resource_post. body: %s"), e.body)
        raise
      message = body.get("message")

    logging.exception(("Exception when calling DefaultApi->"
                       "apis_fqdn_v1_namespaces_namespace_resource_post: %s"),
                      message)
    raise e

  matching = []

  for e in events.items:
    if e.involved_object.uid != uid:
      continue
    matching.append(e)

  return matching


def parse_events(events):
  """Parse events.

  Args:
    events: List of events.

  Returns
    pods_created: Set of unique pod names created.
    services_created: Set of unique services created.
  """
  pattern = re.compile(".*Created.*(pod|Service).*: (.*)", re.IGNORECASE)

  pods = set()
  services = set()
  for e in events:
    m = re.match(pattern, e.message)
    if not m:
      continue

    kind = m.group(1)
    name = m.group(2)

    if kind.lower() == "pod":
      pods.add(name)
    elif kind.lower() == "service":
      services.add(name)

  return pods, services
