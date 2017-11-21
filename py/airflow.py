"""Utilities for working with Airflow."""
import datetime
import json
import logging
import os
import sys
import tempfile
import time

import google.auth
import google.auth.transport.requests
from google.cloud import storage  # pylint: disable=no-name-in-module
from py import prow  # pylint: disable=ungrouped-imports
from py import util  # pylint: disable=ungrouped-imports
import requests

E2E_DAG = "tf_k8s_tests"

# Currently we don't use airflow/api/client because there's not actually much
# functionality in the client. So rather than take a dependency on airflow
# we just copied the code we need. We can revisit that if the Airflow client
# adds more functionality
#
# Note Airflow's client uses urljoin which doesn't produce the expected
# behavior when base_url; see
# https://stackoverflow.com/questions/10893374/python-confusions-with-urljoin
class AirflowClient(object):
  """Extension of the Airflow client."""
  def __init__(self, base_url, credentials=None, verify=True):
    """Construct the client.

    Args:
      base_url: Typically something like https://host:port.
      credentials: A google.auth.credentials object used to obtain bearer
        tokens or None if no credentials are needed
      verify: Whether or not to verify TLS certs
    """
    self._api_base_url = base_url
    self._api_base_url = self._api_base_url.rstrip("/")
    self._credentials = credentials
    self._verify = verify

  def _request(self, url, method="GET", json_body=None):
    params = {
          "url": url,
          "headers": {},
        }
    if json is not None:
      params["json"] = json_body
    params["verify"] = self._verify
    if self._credentials:
      if not self._credentials.valid:
        request = google.auth.transport.requests.Request()
        self._credentials.refresh(request)
      self._credentials.apply(params["headers"])
    resp = getattr(requests, method.lower())(**params)  # pylint: disable=not-callable
    if not resp.ok:
      try:
        data = resp.json()
      except requests.exceptions.RequestException as e:
        logging.error("Failed to request url: %s; error; %s", url, e)
        data = {}
      raise IOError(data.get("error", "Server error"))

    return resp.json()

  def trigger_dag(self, dag_id, run_id=None, conf=None, execution_date=None):
    endpoint = "/api/experimental/dags/{}/dag_runs".format(dag_id)
    url = self._api_base_url + endpoint
    data = self._request(url, method="POST",
                             json_body={
                               "run_id": run_id,
                               "conf": conf,
                               "execution_date": execution_date,
                             })
    return data["message"]

  def get_task_info(self, dag_id, task_id):
    """Return information about the specified task.

    Args:
      dag_id: Id for the DAG
      task_id: Id for the task.

    Returns:
      task_info
    """
    endpoint = "/api/experimental/dags/{0}/tasks/{1}".format(dag_id, task_id)
    url = self._api_base_url + endpoint
    data = self._request(url, method="GET")
    return data

  def get_task_status(self, dag_id, execution_date, task_id):
    """Return information about the specified task.

    Args:
      dag_id: Id for the DAG
      execution_date: String describing the execution date.
      task_id: Id for the task.

    Returns:
      task_info
    """
    endpoint = "/api/experimental/dags/{0}/dag_runs/{1}/tasks/{2}".format(
        dag_id, execution_date, task_id)
    #'/dags/<string:dag_id>/dag_runs/<string:execution_date>/tasks/<string:task_id>
    url = self._api_base_url + endpoint
    data = self._request(url, method="GET")
    return data

def trigger_tf_k8s_tests_dag(client, conf):
  """Trigger a run of tf_k8s_tests pipeline.

  Args:
    client: Airflow client.
    conf: Dictionary containing the conf to parameterize the workflow.

  Returns:
    run_id: The run id for the DAG.
    message: Message returned by the server.
  """
  # We set the run_id to the format expected by get_task_info endpoint
  # See:
  # https://github.com/apache/incubator-airflow/blob/master/airflow/www/api/experimental/endpoints.py#L126
  run_id = datetime.datetime.now().strftime("%Y-%m-%dT%H:%M:%S")

  # TODO(jlewi): It looks like Airflow doesn't actually use run_id as the
  # run_id. I think it ends up using execution date. So if we don't set
  # execution_date the server will end up assigning one. To work around this
  # we set execution date to the run id.
  execution_date = run_id
  resp = client.trigger_dag(E2E_DAG, run_id, json.dumps(conf), execution_date)
  logging.info("Triggered DAG %s run_id %s with conf %s", E2E_DAG, run_id, conf)

  return run_id, resp

def wait_for_tf_k8s_tests(client, run_id,
                          timeout=datetime.timedelta(minutes=30),
                          polling_interval=datetime.timedelta(seconds=15)):
  """Wait for the E2E pipeline to finish.

  Args:
    client: Airflow client.
    run_id: Id of the Airflow run
    timeout: Timeout. Defaults to 20 minutes.
    polling_interval: How often to poll for pipeline status.

  Returns:
    state: The state of the final task.
  """
  endtime = datetime.datetime.now() + timeout
  while True:
    # TODO(jlewi): Airflow only allows us to get the stats of individual tasks
    # not the overall DAG. So we just get the status of the final teardown step.
    # This should be sufficient for our purposes.
    #
    # In the ui it looks like every DAG has a task "undefined" that indicates
    # overall status of the DAG; but we get an error if we try to get this
    # task using the API.
    resp = client.get_task_status(E2E_DAG, run_id, "teardown_cluster")

    state = resp.get("state", "")
    logging.info("State of DAG %s run %s step teardown_cluster: %s",
                 E2E_DAG, run_id, state)
    # If earlier stages fail and teardown_cluster never than the state of
    # of the step will be "upstream_failed"
    if state and not state in ["queued", "running", "None"]:
      return state
    if datetime.datetime.now() + polling_interval > endtime:
      raise util.TimeoutError(
        "Timed out waiting for DAG {0} run {1} to finish.".format(E2E_DAG,
                                                                  run_id))
    logging.info("Waiting for DAG %s run %s to finish.", E2E_DAG, run_id)
    time.sleep(polling_interval.seconds)


def _run_dag_and_wait():
  """Run and wait for the DAG to finish.

  Returns:
    state: The end state of the DAG.
  """
  artifacts_path = os.path.join(prow.get_gcs_output(), "artifacts")
  logging.info("Artifacts will be saved to: %s", artifacts_path)

  conf = {
    "PULL_NUMBER": os.getenv("PULL_NUMBER", ""),
    "PULL_PULL_SHA": os.getenv("PULL_PULL_SHA", ""),
    "PULL_BASE_SHA": os.getenv("PULL_BASE_SHA", ""),
    "ARTIFACTS_PATH": artifacts_path,
  }

  # TODO(jlewi): We should probably configure Ingress and IAP for Airflow sever
  # and use a static IP.
  PROW_K8S_MASTER = "35.202.163.166"
  base_url = ("https://{0}/api/v1/proxy/namespaces/default/services"
              "/airflow:80").format(PROW_K8S_MASTER)

  credentials, _ = google.auth.default(
    scopes=["https://www.googleapis.com/auth/cloud-platform"])

  client = AirflowClient(base_url, credentials, verify=False)

  run_id, _ = trigger_tf_k8s_tests_dag(client, conf)

  state = wait_for_tf_k8s_tests(client, run_id)
  return state

def _print_debug_info():
  """Print out various information to facilitate debugging."""
  this_dir = os.path.dirname(__file__)
  version_file = os.path.join(this_dir, "version.json")
  if os.path.exists(version_file):
    # Print out version information so we know what container we ran in.
    with open(version_file) as hf:
      version = json.load(hf)
      logging.info("Image info:\n%s", json.dumps(version, indent=2,
                                                 sort_keys=True))
  else:
    logging.warn("Could not find file: %s", version_file)

  # Print environment variables.
  # This is useful for debugging and understanding the information set by prow.
  names = os.environ.keys()
  names.sort()
  logging.info("Environment Variables")
  for n in names:
    logging.info("%s=%s", n, os.environ[n])
  logging.info("End Environment Variables")

def main():
  """Trigger Airflow pipelines.

  This main program is intended to be triggered by PROW and used to launch
  The Airflow pipelines comprising our test and release pipelines.
  """
  _print_debug_info()
  # TODO(jlewi): Need to upload various artifacts for gubernator
  # https://github.com/kubernetes/test-infra/tree/master/gubernator.
  # e.g. started.json.
  test_dir = tempfile.mkdtemp(prefix="tmpTfCrdTest")

  # Setup a logging file handler. This file will be the build log.
  rootLogger = logging.getLogger()

  build_log = os.path.join(test_dir, "build-log.txt")
  fileHandler = logging.FileHandler(build_log)
  # TODO(jlewi): Should we set the formatter?
  rootLogger.addHandler(fileHandler)

  logging.info("test_dir: %s", test_dir)

  # Activate the service account for gcloud
  # If you don't activate it then you should already be logged in.
  if os.getenv("GOOGLE_APPLICATION_CREDENTIALS"):
    logging.info("GOOGLE_APPLICATION_CREDENTIALS=%s",
               os.getenv("GOOGLE_APPLICATION_CREDENTIALS"))
    util.run(["gcloud", "auth", "activate-service-account",
            "--key-file={0}".format(os.getenv("GOOGLE_APPLICATION_CREDENTIALS"))])
  job_name = os.getenv("JOB_NAME", "")
  build_number = os.getenv("BUILD_NUMBER")
  pull_number = os.getenv("PULL_NUMBER")
  output_dir = prow.get_gcs_output()

  sha = prow.get_commit_from_env()
  gcs_client = storage.Client()
  prow.create_started(gcs_client, output_dir, sha)

  symlink = prow.get_symlink_output(pull_number, job_name, build_number)
  if symlink:
    prow.create_symlink(gcs_client, symlink, output_dir)

  state = _run_dag_and_wait()

  test_dir = tempfile.mkdtemp(prefix="tmpTfCrdTest")

  success = bool(state == "success")

  if success:
    job_name = os.getenv("JOB_NAME", "unknown")
    prow.create_latest(gcs_client, job_name, sha)

  prow.create_finished(gcs_client, output_dir, success)

  fileHandler.flush()
  prow.upload_outputs(gcs_client, output_dir, build_log)

  if not success:
    # Exit with a non-zero exit code by raising an exception.
    logging.error("One or more test steps failed exiting with non-zero exit "
                  "code.")
    sys.exit(1)

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  main()
