"""Utilities for working with Airflow."""
import datetime
import json
import logging
import os
import requests

import google.auth
import google.auth.transport.requests
from py import util

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
    if self._api_base_url[-1] == "/":
      self._api_base_url = self._api_base_url[:-1]
    self._credentials = credentials
    self._verify = verify

  def _request(self, url, method='GET', json=None):
    params = {
          'url': url,
          "headers": {},
        }
    if json is not None:
      params['json'] = json
    params["verify"] = self._verify
    if self._credentials:
      if not self._credentials.valid:
        request = google.auth.transport.requests.Request()
        self._credentials.refresh(request)
      self._credentials.apply(params["headers"])
    resp = getattr(requests, method.lower())(**params)
    if not resp.ok:
      try:
        data = resp.json()
      except Exception:
        data = {}
      raise IOError(data.get('error', 'Server error'))

    return resp.json()

  def trigger_dag(self, dag_id, run_id=None, conf=None, execution_date=None):
    endpoint = '/api/experimental/dags/{}/dag_runs'.format(dag_id)
    url = self._api_base_url + endpoint
    data = self._request(url, method='POST',
                             json={
                               "run_id": run_id,
                               "conf": conf,
                               "execution_date": execution_date,
                             })
    return data['message']

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
    endpoint = "/api/experimental/dags/{0}/dag_runs/{1}/tasks/{2}".format(dag_id, execution_date, task_id)
    #'/dags/<string:dag_id>/dag_runs/<string:execution_date>/tasks/<string:task_id>
    url = self._api_base_url + endpoint
    print(url)
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
  conf = {
    "PULL_NUMBER": 144,
    "PULL_PULL_SHA": "12e6f092555562628f0c5116ff181a4fbbe1ac35",
  }
  resp = client.trigger_dag(E2E_DAG, run_id, json.dumps(conf), execution_date)

  return run_id, resp

def wait_for_tf_k8s_tests(client, run_id,
                          timeout=datetime.timedelta(minutes=20),
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
    resp = client.get_task_status(E2E_DAG, run_id, "teardown_cluster")

    state = resp.get("state", "")

    # TODO(jlewi): What if earlier stages fail and teardown_cluster never
    # runs? Perhaps we should look at every task in the DAG.
    if state and not state == "running":
      return state
    if datetime.datetime.now() + polling_interval > endtime:
      raise util.TimeoutError(
        "Timed out waiting for DAG {0} run {1} to finish.".format(E2E_DAG,
                                                                  run_id))
    logging.info("Waiting for DAG %s run %s to finish.", E2E_DAG, run_id)
    time.sleep(polling_interval.seconds)

def clone_repo():
  """Clone the repo.

  Returns:
    src_path: This is the root path for the training code.
    sha: The sha number of the repo.
  """
  # If this is a presubmit PULL_PULL_SHA will be set see:
  # https://github.com/kubernetes/test-infra/tree/master/prow#job-evironment-variables
  sha = ""
  pull_number = os.getenv("PULL_NUMBER", "")

  if pull_number:
    sha = os.getenv("PULL_PULL_SHA", "")
  else:
    # For postsubmits PULL_BASE_SHA will be set
    sha = os.getenv("PULL_BASE_SHA", "")

  if sha:
    run(["git", "checkout", sha], cwd=dest)

  # Get the actual git hash.
  # This ensures even for periodic jobs which don't set the sha we know
  # the version of the code tested.
  sha = run_and_output(["git", "rev-parse", "HEAD"], cwd=dest)

  # Install dependencies
  # TODO(jlewi): We should probably move this into runner.py so that
  # output is captured in build-log.txt
  run(["glide", "install"], cwd=dest)

  # We need to remove the vendored dependencies of apiextensions otherwise we
  # get type conflicts.
  shutil.rmtree(os.path.join(dest,
                             "vendor/k8s.io/apiextensions-apiserver/vendor"))

  return dest, sha


def main():
  """Trigger Airflow pipelines.

  This main program is intended to be triggered by PROW and used to launch
  The Airflow pipelines comprising our test and release pipelines.
  """
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

  conf = {
    "PULL_NUMBER": os.getenv("PULL_NUMBER", ""),
    "PULL_PULL_SHA": os.getenv("PULL_PULL_SHA", ""),
    "PULL_BASE_SHA": os.getenv("PULL_BASE_SHA", ""),
  }

  # TODO(jlewi): We should probably configure Ingress and IAP for Airflow sever
  # and use a static IP.
  PROW_K8S_MASTER = "35.202.163.166"
  base_url = "https://{0}/api/v1/proxy/namespaces/default/services/airflow:80".format(PROW_K8S_MASTER)

  credentials, _ = google.auth.default(
    scopes=["https://www.googleapis.com/auth/cloud-platform"])

  client = AirflowClient(base_url, credentials, verify=False)

  run_id, message = trigger_tf_k8s_tests_dag(client, conf)
  logging.info("Triggered DAG %s run_id %s".format(E2E_DAG, run_id))

  state = wait_for_tf_k8s_tests(client, run_id)
  logging.info("DAG %s run_id %s is in state %s".format(E2E_DAG, run_id, state))

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  main()
