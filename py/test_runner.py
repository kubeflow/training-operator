"""Test runner runs a TFJob test."""

import argparse
import datetime
import logging
import json
import os
import retrying
import time
import yaml

from kubernetes import client as k8s_client

from google.cloud import storage  # pylint: disable=no-name-in-module
from kubeflow.testing import util
from py import k8s_util
from py import ks_util
from py import test_util
from py import tf_job_client
from py import util as tf_operator_util


def get_runconfig(master_host, namespace, target):
  """Issue a request to get the runconfig of the specified replica running test_server.

    Args:
    master_host: The IP address of the master e.g. https://35.188.37.10
    namespace: The namespace
    target: The K8s service corresponding to the pod to call.
  """
  response = tf_operator_util.send_request(master_host, namespace, target, "runconfig", {})
  return yaml.load(response)


def verify_runconfig(master_host, namespace, job_name, replica, num_ps, num_workers):
  """Verifies that the TF RunConfig on the specified replica is the same as expected.

    Args:
    master_host: The IP address of the master e.g. https://35.188.37.10
    namespace: The namespace
    job_name: The name of the TF job
    replica: The replica type (chief, ps, or worker)
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

  # Construct the expected cluster spec
  chief_list = ["{name}-chief-0:2222".format(name=job_name)]
  ps_list = []
  for i in range(num_ps):
    ps_list.append("{name}-ps-{index}:2222".format(name=job_name, index=i))
  worker_list = []
  for i in range(num_workers):
    worker_list.append("{name}-worker-{index}:2222".format(name=job_name, index=i))
  cluster_spec = {
    "chief": chief_list,
    "ps": ps_list,
    "worker": worker_list,
  }

  for i in range(num_replicas):
    full_target = "{name}-{replica}-{index}".format(name=job_name, replica=replica.lower(), index=i)
    actual_config = get_runconfig(master_host, namespace, full_target)
    expected_config = {
      "task_type": replica,
      "task_id": i,
      "cluster_spec": cluster_spec,
      "is_chief": is_chief,
      "master": "grpc://{target}:2222".format(target=full_target),
      "num_worker_replicas": num_workers + 1, # Chief is also a worker
      "num_ps_replicas": num_ps,
    }
    # Compare expected and actual configs
    if actual_config != expected_config:
      msg = "Actual runconfig differs from expected. Expected: {0} Actual: {1}".format(
        str(expected_config), str(actual_config))
      logging.error(msg)
      raise RuntimeError(msg)


# One of the reasons we set so many retries and a random amount of wait
# between retries is because we have multiple tests running in parallel
# that are all modifying the same ksonnet app via ks. I think this can
# lead to failures.
@retrying.retry(stop_max_attempt_number=10, wait_random_min=1000,
                wait_random_max=10000)
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
  masterHost = api_client.configuration.host

  t = test_util.TestCase()
  t.class_name = "tfjob_test"
  namespace, name, env = ks_util.setup_ks_app(args)
  t.name = os.path.basename(name)

  start = time.time()


  try: # pylint: disable=too-many-nested-blocks
    # We repeat the test multiple times.
    # This ensures that if we delete the job we can create a new job with the
    # same name.

    # TODO(jlewi): We should make this an argument.
    num_trials = 2

    for trial in range(num_trials):
      logging.info("Trial %s", trial)
      util.run(["ks", "apply", env, "-c", args.component], cwd=args.app_dir)

      logging.info("Created job %s in namespaces %s", name, namespace)
      logging.info("tfjob_version=%s", args.tfjob_version)
      # Wait for the job to either be in Running state or a terminal state
      if args.tfjob_version == "v1alpha1":
        logging.info("Wait for Phase Running, Done, or Failed")
        results = tf_job_client.wait_for_phase(
          api_client, namespace, name, ["Running", "Done", "Failed"],
          status_callback=tf_job_client.log_status)
      else:
        logging.info("Wait for conditions Running, Succeeded, or Failed")
        results = tf_job_client.wait_for_condition(
          api_client, namespace, name, ["Running", "Succeeded", "Failed"],
          status_callback=tf_job_client.log_status)

      logging.info("Current TFJob:\n %s", json.dumps(results, indent=2))

      # The job is now either running or done.
      if args.shutdown_policy:
        logging.info("Enforcing shutdownPolicy %s", args.shutdown_policy)
        if args.shutdown_policy in ["master", "chief"]:
          if args.tfjob_version == "v1alpha1":
            replica = "master"
          else:
            replica = "chief"
        elif args.shutdown_policy in ["worker", "all_workers"]:
          replica = "worker"
        else:
          raise ValueError("Unrecognized shutdown_policy "
                           "%s" % args.shutdown_policy)

        # Number of targets.
        num_targets = 1
        if args.shutdown_policy in ["all_workers"]:
          # Assume v1alpha2
          num_targets = results.get("spec", {}).get("tfReplicaSpecs", {}).get(
            "Worker", {}).get("replicas", 0)
          logging.info("There are %s worker replicas", num_targets)


        if args.tfjob_version == "v1alpha1":
          runtime_id = results.get("spec", {}).get("RuntimeId")
          target = "{name}-{replica}-{runtime}".format(
            name=name, replica=replica, runtime=runtime_id)
          pod_labels = tf_job_client.get_labels(name, runtime_id)
          pod_selector = tf_job_client.to_selector(pod_labels)
        else:
          target = "{name}-{replica}".format(name=name, replica=replica)
          pod_labels = tf_job_client.get_labels_v1alpha2(namespace, name)
          pod_selector = tf_job_client.to_selector(pod_labels)

        # Wait for the pods to be ready before we shutdown
        # TODO(jlewi): We are get pods using a label selector so there is
        # a risk that the pod we actual care about isn't present.
        logging.info("Waiting for pods to be running before shutting down.")
        k8s_util.wait_for_pods_to_be_in_phases(api_client, namespace,
                                               pod_selector,
                                               ["Running"],
                                               timeout=datetime.timedelta(
                                               minutes=4))
        logging.info("Pods are ready")
        logging.info("Issuing the terminate request")
        for num in range(num_targets):
          full_target = target + "-{0}".format(num)
          tf_job_client.terminate_replica(masterHost, namespace, full_target)

      # TODO(richardsliu):
      # There are lots of verifications in this file, consider refactoring them.
      if args.verify_runconfig:
        num_ps = results.get("spec", {}).get("tfReplicaSpecs", {}).get(
          "PS", {}).get("replicas", 0)
        num_workers = results.get("spec", {}).get("tfReplicaSpecs", {}).get(
          "Worker", {}).get("replicas", 0)
        verify_runconfig(masterHost, namespace, name, "chief", num_ps, num_workers)
        verify_runconfig(masterHost, namespace, name, "worker", num_ps, num_workers)
        verify_runconfig(masterHost, namespace, name, "ps", num_ps, num_workers)

        # Terminate the chief worker to complete the job.
        tf_job_client.terminate_replica(masterHost, namespace, "{name}-chief-0".format(name=name))

      logging.info("Waiting for job to finish.")
      results = tf_job_client.wait_for_job(
        api_client, namespace, name, args.tfjob_version,
        status_callback=tf_job_client.log_status)

      logging.info("Final TFJob:\n %s", json.dumps(results, indent=2))

      if args.tfjob_version == "v1alpha1":
        if results.get("status", {}).get("state", {}).lower() != "succeeded":
          t.failure = "Trial {0} Job {1} in namespace {2} in state {3}".format(
            trial, name, namespace, results.get("status", {}).get("state", None))
          logging.error(t.failure)
          break
      else:
        # For v1alpha2 check for non-empty completionTime
        last_condition = results.get("status", {}).get("conditions", [])[-1]
        if last_condition.get("type", "").lower() != "succeeded":
          t.failure = "Trial {0} Job {1} in namespace {2} in status {3}".format(
            trial, name, namespace, results.get("status", {}))
          logging.error(t.failure)
          break

      runtime_id = results.get("spec", {}).get("RuntimeId")
      logging.info("Trial %s Job %s in namespace %s runtime ID %s", trial, name,
                   namespace, runtime_id)

      uid = results.get("metadata", {}).get("uid")
      events = k8s_util.get_events(api_client, namespace, uid)
      for e in events:
        logging.info("K8s event: %s", e.message)

      # Print out the K8s events because it can be useful for debugging.
      for e in events:
        logging.info("Recieved K8s Event:\n%s", e)
      created_pods, created_services = k8s_util.parse_events(events)

      num_expected = 0
      if args.tfjob_version == "v1alpha1":
        for replica in results.get("spec", {}).get("replicaSpecs", []):
          num_expected += replica.get("replicas", 0)
      else:
        for replicakey in results.get("spec", {}).get("tfReplicaSpecs", {}):
          replica_spec = results.get("spec", {}).get("tfReplicaSpecs", {}).get(replicakey, {})
          if replica_spec:
            num_expected += replica_spec.get("replicas", 1)

      creation_failures = []
      if len(created_pods) != num_expected:
        message = ("Expected {0} pods to be created but only "
                   "got {1} create events.").format(num_expected,
                                                    len(created_pods))
        creation_failures.append(message)

      if len(created_services) != num_expected:
        message = ("Expected {0} services to be created but only "
                   "got {1} create events.").format(num_expected,
                                                    len(created_services))
        creation_failures.append(message)

      if creation_failures:
        # TODO(jlewi): Starting with
        # https://github.com/kubeflow/tf-operator/pull/646 the number of events
        # no longer seems to match the expected; it looks like maybe events
        # are being combined? For now we just log a warning rather than an
        # error.
        logging.warning(creation_failures)
      if args.tfjob_version == "v1alpha1":
        pod_labels = tf_job_client.get_labels(name, runtime_id)
        pod_selector = tf_job_client.to_selector(pod_labels)
      else:
        pod_labels = tf_job_client.get_labels_v1alpha2(name)
        pod_selector = tf_job_client.to_selector(pod_labels)

      # In v1alpha1 all pods are deleted. In v1alpha2, this depends on the pod
      # cleanup policy.
      if args.tfjob_version == "v1alpha1":
        k8s_util.wait_for_pods_to_be_deleted(api_client, namespace, pod_selector)
      else:
        # All pods are deleted.
        if args.verify_clean_pod_policy == "All":
          k8s_util.wait_for_pods_to_be_deleted(api_client, namespace, pod_selector)
        # Only running pods (PS) are deleted, completed pods are not.
        elif args.verify_clean_pod_policy == "Running":
          tf_job_client.wait_for_replica_type_in_phases(api_client, namespace,
                                                        name, "Chief", ["Completed"])
          tf_job_client.wait_for_replica_type_in_phases(api_client, namespace, name,
                                                        "Worker", ["Completed"])
          ps_pod_labels = tf_job_client.get_labels_v1alpha2(name, "PS")
          ps_pod_selector = tf_job_client.to_selector(ps_pod_labels)
          k8s_util.wait_for_pods_to_be_deleted(api_client, namespace, ps_pod_selector)
        # No pods are deleted.
        elif args.verify_clean_pod_policy == "None":
          tf_job_client.wait_for_replica_type_in_phases(api_client, namespace,
                                                        name, "Chief", ["Completed"])
          tf_job_client.wait_for_replica_type_in_phases(api_client, namespace,
                                                        name, "Worker", ["Completed"])
          tf_job_client.wait_for_replica_type_in_phases(api_client, namespace,
                                                        name, "PS", ["Running"])

      tf_job_client.delete_tf_job(api_client, namespace, name, version=args.tfjob_version)

      logging.info("Waiting for job %s in namespaces %s to be deleted.", name,
                   namespace)
      k8s_util.wait_for_delete(
        api_client, namespace, name, args.tfjob_version, status_callback=tf_job_client.log_status)

    # TODO(jlewi):
    #  Here are some validation checks to run:
    #  1. Check that all resources are garbage collected.
    # TODO(jlewi): Add an option to add chaos and randomly kill various resources?
    # TODO(jlewi): Are there other generic validation checks we should
    # run.
  except tf_operator_util.JobTimeoutError as e:
    if e.job:
      spec = "Job:\n" + json.dumps(e.job, indent=2)
    else:
      spec = "JobTimeoutError did not contain job"
    t.failure = ("Timeout waiting for {0} in namespace {1} to finish; ").format(
                  name, namespace) + spec
    logging.exception(t.failure)
  except Exception as e:  # pylint: disable-msg=broad-except
    # TODO(jlewi): I'm observing flakes where the exception has message "status"
    # in an effort to try to nail down this exception we print out more
    # information about the exception.
    logging.exception("There was a problem running the job; Exception %s", e)
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
    "--shutdown_policy",
    default=None,
    type=str,
    help="The shutdown policy. This must be set if we need to issue "
         "an http request to the test-app server to exit before the job will "
         "finish.")

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

  parser.add_argument(
    "--tfjob_version",
    default="v1alpha1",
    type=str,
    help="The TFJob version to use.")

  parser.add_argument(
    "--environment",
    default=None,
    type=str,
    help="(Optional) the name for the ksonnet environment; if not specified "
         "a random one is created.")

  parser.add_argument(
    "--verify_clean_pod_policy",
    default=None,
    type=str,
    help="(Optional) the clean pod policy (None, Running, or All).")

  parser.add_argument(
    "--verify_runconfig",
    dest="verify_runconfig",
    action="store_true",
    help="(Optional) verify runconfig in each replica.")


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
    datefmt='%Y-%m-%dT%H:%M:%S',)

  util.maybe_activate_service_account()

  parser = build_parser()

  # parse the args and call whatever function was selected
  args = parser.parse_args()
  args.func(args)


if __name__ == "__main__":
  main()
