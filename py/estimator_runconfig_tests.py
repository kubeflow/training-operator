import json
import logging
import yaml
from kubernetes import client as k8s_client
from kubeflow.testing import ks_util
from kubeflow.testing import test_util, util
from py import test_runner
from py import tf_job_client
from py import util as tf_operator_util

COMPONENT_NAME = "estimator_runconfig"


def get_runconfig(master_host, namespace, target):
  """Issue a request to get the runconfig of the specified replica running test_server.
    Args:
    master_host: The IP address of the master e.g. https://35.188.37.10
    namespace: The namespace
    target: The K8s service corresponding to the pod to call.
  """
  response = tf_operator_util.send_request(master_host, namespace, target,
                                           "runconfig", {})
  return yaml.load(response)


def verify_runconfig(master_host, namespace, job_name, replica, num_ps,
                     num_workers):
  """Verifies that the TF RunConfig on the specified replica is the same as expected.
    Args:
    master_host: The IP address of the master e.g. https://35.188.37.10
    namespace: The namespace
    job_name: The name of the TF job
    replica: The replica type (chief, ps, worker, or evaluator)
    num_ps: The number of PS replicas
    num_workers: The number of worker replicas
  """
  is_chief = True
  num_replicas = 1
  if replica == "ps":
    is_chief = False
    num_replicas = num_ps
  elif replica == "worker":
    is_chief = False
    num_replicas = num_workers
  elif replica == "evaluator":
    is_chief = False

  # Construct the expected cluster spec
  chief_list = ["{name}-chief-0.{ns}.svc:2222".format(name=job_name, ns=namespace)]
  ps_list = []
  for i in range(num_ps):
    ps_list.append("{name}-ps-{index}.{ns}.svc:2222".format(name=job_name, index=i, ns=namespace))
  worker_list = []
  for i in range(num_workers):
    worker_list.append("{name}-worker-{index}.{ns}.svc:2222".format(name=job_name,
      index=i, ns=namespace))
  # Evaluator is not part of training cluster.
  cluster_spec = {
    "chief": chief_list,
    "ps": ps_list,
    "worker": worker_list,
  }

  for i in range(num_replicas):
    full_target = "{name}-{replica}-{index}".format(
      name=job_name, replica=replica.lower(), index=i)
    actual_config = get_runconfig(master_host, namespace, full_target)
    full_svc = "{ft}.{ns}.svc".format(ft=full_target, ns=namespace)
    expected_config = {
      "task_type": replica,
      "task_id": i,
      "cluster_spec": cluster_spec,
      "is_chief": is_chief,
      "master": "grpc://{fs}:2222".format(fs=full_svc),
      "num_worker_replicas": num_workers + 1, # Chief is also a worker
      "num_ps_replicas": num_ps,
    } if not replica == "evaluator" else {
      # Evaluator has special config.
      "task_type": replica,
      "task_id": 0,
      "cluster_spec": {},
      "is_chief": is_chief,
      "master": "",
      "num_worker_replicas": 0,
      "num_ps_replicas": 0,
    }

    # Compare expected and actual configs
    if actual_config != expected_config:
      msg = "Actual runconfig differs from expected. Expected: {0} Actual: {1}".format(
        str(expected_config), str(actual_config))
      logging.error(msg)
      raise RuntimeError(msg)


class EstimatorRunconfigTests(test_util.TestCase):

  def __init__(self, args):
    namespace, name, env = test_runner.parse_runtime_params(args)
    self.app_dir = args.app_dir
    self.env = env
    self.namespace = namespace
    self.tfjob_version = args.tfjob_version
    self.params = args.params
    super(EstimatorRunconfigTests, self).__init__(
      class_name="EstimatorRunconfigTests", name=name)

  # Run a TFJob, verify that the TensorFlow runconfig specs are set correctly.
  def test_tfjob_and_verify_runconfig(self):
    api_client = k8s_client.ApiClient()
    masterHost = api_client.configuration.host
    component = COMPONENT_NAME + "_" + self.tfjob_version

    # Setup the ksonnet app
    ks_util.setup_ks_app(self.app_dir, self.env, self.namespace, component,
                         self.params)

    # Create the TF job
    util.run(["ks", "apply", self.env, "-c", component], cwd=self.app_dir)
    logging.info("Created job %s in namespaces %s", self.name, self.namespace)

    # Wait for the job to either be in Running state or a terminal state
    logging.info("Wait for conditions Running, Succeeded, or Failed")
    results = tf_job_client.wait_for_condition(
      api_client,
      self.namespace,
      self.name, ["Running", "Succeeded", "Failed"],
      version=self.tfjob_version,
      status_callback=tf_job_client.log_status)
    logging.info("Current TFJob:\n %s", json.dumps(results, indent=2))

    num_ps = results.get("spec", {}).get("tfReplicaSpecs",
                                         {}).get("PS", {}).get("replicas", 0)
    num_workers = results.get("spec", {}).get("tfReplicaSpecs", {}).get(
      "Worker", {}).get("replicas", 0)
    verify_runconfig(masterHost, self.namespace, self.name, "chief", num_ps,
                     num_workers)
    verify_runconfig(masterHost, self.namespace, self.name, "worker", num_ps,
                     num_workers)
    verify_runconfig(masterHost, self.namespace, self.name, "ps", num_ps,
                     num_workers)
    verify_runconfig(masterHost, self.namespace, self.name, "evaluator", num_ps,
                     num_workers)

    tf_job_client.terminate_replicas(api_client, self.namespace, self.name,
                                     "chief", 1)

    # Wait for the job to complete.
    logging.info("Waiting for job to finish.")
    results = tf_job_client.wait_for_job(
      api_client,
      self.namespace,
      self.name,
      self.tfjob_version,
      status_callback=tf_job_client.log_status)
    logging.info("Final TFJob:\n %s", json.dumps(results, indent=2))

    if not tf_job_client.job_succeeded(results):
      self.failure = "Job {0} in namespace {1} in status {2}".format(
        self.name, self.namespace, results.get("status", {}))
      logging.error(self.failure)

    # Delete the TFJob.
    tf_job_client.delete_tf_job(
      api_client, self.namespace, self.name, version=self.tfjob_version)
    logging.info("Waiting for job %s in namespaces %s to be deleted.",
                 self.name, self.namespace)
    tf_job_client.wait_for_delete(
      api_client,
      self.namespace,
      self.name,
      self.tfjob_version,
      status_callback=tf_job_client.log_status)


if __name__ == "__main__":
  test_runner.main(module=__name__)
