"""
An Airflow pipeline for running our E2E tests.
"""

# TODO(jlewi): We should split setup into two steps; create cluster and setup
# cluster. The cluster can be created in parallel with building the artifacts
# which should speed things up.

from datetime import datetime
import logging
import os
import tempfile
import uuid

from airflow import DAG
from airflow.operators import PythonOperator
from py import util
from google.cloud import storage  # pylint: disable=no-name-in-module
import six
import yaml

default_args = {
  'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime(2015, 6, 1),
    'email': ['airflow@airflow.com'],
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
}

dag = DAG(
  # Set schedule_interval to None
  'tf_k8s_tests', default_args=default_args,
  # TODO(jlewi): Should we schedule a regular run? Right now its
  # manually triggered by PROW.
  schedule_interval=None)

# Default name for the repo organization and name.
# This should match the values used in Go imports.
DEFAULT_REPO_OWNER = "tensorflow"
DEFAULT_REPO_NAME = "k8s"

# GCS path to use for each dag run
GCS_RUNS_PATH = "gs://mlkube-testing-airflow/runs"

GCB_PROJECT = "mlkube-testing"
ZONE = "us-east1-d"

def run_path(dag_id, run_id):
  return os.path.join(GCS_RUNS_PATH, dag_id.replace(":", "_"),
                      run_id.replace(":", "_"))

class FakeDagrun(object):
  def __init__(self):
    self.dag_id = "tf_k8s_tests"
    self.run_id = "test_run"
    self.conf = {}

def clone_repo(dag_run=None, ti=None, **_kwargs): # pylint: disable=too-many-statements
  # Create a temporary directory suitable for checking out and building the
  # code.
  if not dag_run:
    # When running via airflow test dag_run isn't set
    logging.warn("Using fake dag_run")
    dag_run = FakeDagrun()

  logging.info("dag_id: %s", dag_run.dag_id)
  logging.info("run_id: %s", dag_run.run_id)

  conf = dag_run.conf
  if not conf:
    conf = {}
  logging.info("conf=%s", conf)


  # Pick the directory the top level directory to use for this run of the
  # pipeline.
  # This should be a persistent location that is accessible from subsequent
  # tasks; e.g. an NFS share or PD. The environment variable SRC_DIR is used
  # to allow the directory to be specified as part of the deployment
  run_dir = os.path.join(os.getenv("SRC_DIR", tempfile.gettempdir()),
                         dag_run.dag_id.replace(":", "_"),
                         dag_run.run_id.replace(":", "_"))
  logging.info("Using run_dir %s", run_dir)
  os.makedirs(run_dir)
  logging.info("xcom push: run_dir=%s", run_dir)
  ti.xcom_push(key="run_dir", value=run_dir)

  # Directory where we will clone the src
  src_dir = os.path.join(run_dir, "tensorflow_k8s")
  logging.info("xcom push: src_dir=%s", src_dir)
  ti.xcom_push(key="src_dir", value=src_dir)

  # Make sure pull_number is a string
  pull_number = "{0}".format(conf.get("PULL_NUMBER", ""))
  args = ["python", "-m", "py.release", "clone", "--src_dir=" + src_dir]
  if pull_number:
    commit = conf.get("PULL_PULL_SHA", "")
    args.append("pr")
    args.append("--pr=" + pull_number)
    if commit:
      args.append("--commit=" + commit)
  else:
    commit = conf.get("PULL_BASE_SHA", "")
    args.append("postsubmit")
    if commit:
      args.append("--commit=" + commit)

  # The directory baked into the container that contains a copy of py.release
  # that can be used to clone the repo.
  bootstrap_dir = "/opt/tensorflow_k8s"
  util.run(args, cwd=bootstrap_dir, use_print=True)

def build_images(dag_run=None, ti=None, **_kwargs): # pylint: disable=too-many-statements
  """
  Args:
    dag_run: A DagRun object. This is passed in as a result of setting
      provide_context to true for the operator.
  """
  # Create a temporary directory suitable for checking out and building the
  # code.
  if not dag_run:
    # When running via airflow test dag_run isn't set
    logging.warn("Using fake dag_run")
    dag_run = FakeDagrun()

  logging.info("dag_id: %s", dag_run.dag_id)
  logging.info("run_id: %s", dag_run.run_id)

  run_dir = ti.xcom_pull(None, key="run_dir")
  logging.info("Using run_dir=%s", run_dir)

  src_dir = ti.xcom_pull(None, key="src_dir")
  logging.info("Using src_dir=%s", src_dir)

  gcs_path = run_path(dag_run.dag_id, dag_run.run_id)
  logging.info("gcs_path %s", gcs_path)

  conf = dag_run.conf
  if not conf:
    conf = {}
  logging.info("conf=%s", conf)
  artifacts_path = conf.get("ARTIFACTS_PATH", gcs_path)
  logging.info("artifacts_path %s", artifacts_path)

  # We use a GOPATH that is specific to this run because we don't want
  # interference from different runs.
  newenv = os.environ.copy()
  newenv["GOPATH"] = os.path.join(run_dir, "go")

  # Make sure pull_number is a string
  args = ["python", "-m", "py.release", "build", "--src_dir=" + src_dir]

  dryrun = bool(conf.get("dryrun", False))

  build_info_file = os.path.join(gcs_path, "build_info.yaml")
  args.append("--build_info_path=" + build_info_file)
  args.append("--releases_path=" + gcs_path)
  args.append("--project=" + GCB_PROJECT)
  # We want subprocess output to bypass logging module otherwise multiline
  # output is squashed together.
  util.run(args, cwd=src_dir, use_print=True, dryrun=dryrun, env=newenv)

  # Read the output yaml and publish relevant values to xcom.
  if not dryrun:
    gcs_client = storage.Client(project=GCB_PROJECT)
    logging.info("Reading %s", build_info_file)
    bucket_name, build_path = util.split_gcs_uri(build_info_file)
    bucket = gcs_client.get_bucket(bucket_name)
    blob = bucket.blob(build_path)
    contents = blob.download_as_string()
    build_info = yaml.load(contents)
  else:
    build_info = {
      "image": "gcr.io/dryrun/dryrun:latest",
      "commit": "1234abcd",
      "helm_chart": "gs://dryrun/dryrun.latest.",
    }
  for k, v in six.iteritems(build_info):
    logging.info("xcom push: %s=%s", k, v)
    ti.xcom_push(key=k, value=v)

def setup_cluster(dag_run=None, ti=None, **_kwargs):
  conf = dag_run.conf
  if not conf:
    conf = {}

  dryrun = bool(conf.get("dryrun", False))

  chart = ti.xcom_pull("build_images", key="helm_chart")

  now = datetime.now()
  cluster = "e2e-" + now.strftime("%m%d-%H%M-") + uuid.uuid4().hex[0:4]

  logging.info("conf=%s", conf)
  artifacts_path = conf.get("ARTIFACTS_PATH",
                            run_path(dag_run.dag_id, dag_run.run_id))
  logging.info("artifacts_path %s", artifacts_path)

  # Gubernator only recognizes XML files whos name matches
  # junit_[^_]*.xml which is why its "setupcluster" and not "setup_cluster"
  junit_path = os.path.join(artifacts_path, "junit_setupcluster.xml")
  logging.info("junit_path %s", junit_path)

  args = ["python", "-m", "py.deploy", "setup"]
  args.append("--cluster=" + cluster)
  args.append("--junit_path=" + junit_path)
  args.append("--project=" + GCB_PROJECT)
  args.append("--chart=" + chart)
  args.append("--zone=" + ZONE)
  args.append("--accelerator=nvidia-tesla-k80=1")
  # We want subprocess output to bypass logging module otherwise multiline
  # output is squashed together.
  src_dir = ti.xcom_pull(None, key="src_dir")
  util.run(args, cwd=src_dir, use_print=True, dryrun=dryrun)

  values = {
    "cluster": cluster,
  }
  for k, v in six.iteritems(values):
    logging.info("xcom push: %s=%s", k, v)
    ti.xcom_push(key=k, value=v)

def run_tests(dag_run=None, ti=None, **_kwargs):
  conf = dag_run.conf
  if not conf:
    conf = {}

  dryrun = bool(conf.get("dryrun", False))

  cluster = ti.xcom_pull("setup_cluster", key="cluster")

  logging.info("conf=%s", conf)
  artifacts_path = conf.get("ARTIFACTS_PATH",
                            run_path(dag_run.dag_id, dag_run.run_id))
  logging.info("artifacts_path %s", artifacts_path)
  junit_path = os.path.join(artifacts_path, "junit_e2e.xml")
  logging.info("junit_path %s", junit_path)
  ti.xcom_push(key="cluster", value=cluster)

  args = ["python", "-m", "py.deploy", "test"]
  args.append("--cluster=" + cluster)
  args.append("--junit_path=" + junit_path)
  args.append("--project=" + GCB_PROJECT)

  # We want subprocess output to bypass logging module otherwise multiline
  # output is squashed together.
  src_dir = ti.xcom_pull(None, key="src_dir")
  util.run(args, cwd=src_dir, use_print=True, dryrun=dryrun)

# TODO(jlewi): We should make this a function that will generate a callable
# for different configs like we do for py_checks.
def run_gpu_test(dag_run=None, ti=None, **_kwargs):
  conf = dag_run.conf
  if not conf:
    conf = {}

  cluster = ti.xcom_pull(None, key="cluster")

  src_dir = ti.xcom_pull(None, key="src_dir")

  logging.info("conf=%s", conf)
  artifacts_path = conf.get("ARTIFACTS_PATH",
                            run_path(dag_run.dag_id, dag_run.run_id))
  logging.info("artifacts_path %s", artifacts_path)
  # I think we can only have one underscore in the name for gubernator to
  # work.
  junit_path = os.path.join(artifacts_path, "junit_gpu-tests.xml")
  logging.info("junit_path %s", junit_path)
  ti.xcom_push(key="cluster", value=cluster)

  spec = os.path.join(src_dir, "examples/tf_job_gpu.yaml")
  args = ["python", "-m", "py.test_runner", "test"]
  args.append("--spec=" + spec)
  args.append("--zone=" + ZONE)
  args.append("--cluster=" + cluster)
  args.append("--junit_path=" + junit_path)
  args.append("--project=" + GCB_PROJECT)
  # tf_job_gpu.yaml has the image tag hardcoded so the tag doesn't matter.
  # TODO(jlewi): The example should be a template and we should rebuild and
  # and use the newly built sample container.
  args.append("--image_tag=notag")

  # We want subprocess output to bypass logging module otherwise multiline
  # output is squashed together.
  util.run(args, use_print=True, cwd=src_dir)

def teardown_cluster(dag_run=None, ti=None, **_kwargs):
  conf = dag_run.conf
  if not conf:
    conf = {}

  dryrun = bool(conf.get("dryrun", False))

  cluster = ti.xcom_pull("setup_cluster", key="cluster")

  gcs_path = run_path(dag_run.dag_id, dag_run.run_id)

  artifacts_path = conf.get("ARTIFACTS_PATH", gcs_path)
  logging.info("artifacts_path %s", artifacts_path)

  junit_path = os.path.join(artifacts_path, "junit_teardown.xml")
  logging.info("junit_path %s", junit_path)
  ti.xcom_push(key="cluster", value=cluster)

  args = ["python", "-m", "py.deploy", "teardown"]
  args.append("--cluster=" + cluster)
  args.append("--junit_path=" + junit_path)
  args.append("--project=" + GCB_PROJECT)

  # We want subprocess output to bypass logging module otherwise multiline
  # output is squashed together.
  src_dir = ti.xcom_pull(None, key="src_dir")
  util.run(args, cwd=src_dir, use_print=True, dryrun=dryrun)

def py_checks_gen(command):
  """Create a callable to run the specified py_check command."""
  def run_py_checks(dag_run=None, ti=None, **_kwargs):
    """Run some of the python checks."""

    conf = dag_run.conf
    if not conf:
      conf = {}

    dryrun = bool(conf.get("dryrun", False))

    src_dir = ti.xcom_pull(None, key="src_dir")
    logging.info("src_dir %s", src_dir)

    gcs_path = run_path(dag_run.dag_id, dag_run.run_id)

    artifacts_path = conf.get("ARTIFACTS_PATH", gcs_path)
    logging.info("artifacts_path %s", artifacts_path)

    junit_path = os.path.join(artifacts_path, "junit_pychecks{0}.xml".format(command))
    logging.info("junit_path %s", junit_path)

    args = ["python", "-m", "py.py_checks", command]
    args.append("--src_dir=" + src_dir)
    args.append("--junit_path=" + junit_path)
    args.append("--project=" + GCB_PROJECT)

    # We want subprocess output to bypass logging module otherwise multiline
    # output is squashed together.
    util.run(args, cwd=src_dir, use_print=True, dryrun=dryrun)

  return run_py_checks

def done(**_kwargs):
  logging.info("Executing done step.")

clone_op = PythonOperator(
  task_id='clone_repo',
    provide_context=True,
    python_callable=clone_repo,
    dag=dag)

build_op = PythonOperator(
  task_id='build_images',
    provide_context=True,
    python_callable=build_images,
    dag=dag)
build_op.set_upstream(clone_op)

py_lint_op = PythonOperator(
  task_id='pylint',
    provide_context=True,
    python_callable=py_checks_gen("lint"),
    dag=dag)
py_lint_op.set_upstream(clone_op)

py_test_op = PythonOperator(
  task_id='pytest',
    provide_context=True,
    python_callable=py_checks_gen("test"),
    dag=dag)
py_test_op.set_upstream(clone_op)

setup_cluster_op = PythonOperator(
  task_id='setup_cluster',
    provide_context=True,
    python_callable=setup_cluster,
    dag=dag)

setup_cluster_op.set_upstream(build_op)

run_tests_op = PythonOperator(
  task_id='run_tests',
    provide_context=True,
    python_callable=run_tests,
    dag=dag)

run_tests_op.set_upstream(setup_cluster_op)

run_gpu_test_op = PythonOperator(
  task_id='run_gpu_test',
    provide_context=True,
    python_callable=run_gpu_test,
    dag=dag)

run_gpu_test_op.set_upstream(setup_cluster_op)

teardown_cluster_op = PythonOperator(
  task_id='teardown_cluster',
    provide_context=True,
    python_callable=teardown_cluster,
    # We want to trigger the teardown step even if the previous steps failed
    # because we don't want to leave clusters up.
    trigger_rule="all_done",
    dag=dag)

teardown_cluster_op.set_upstream([run_tests_op, run_gpu_test_op])

# Create an op that can be used to determine when all other tasks are done.
done_op = PythonOperator(
  task_id='done',
    provide_context=True,
    python_callable=done,
    dag=dag)

done_op.set_upstream([teardown_cluster_op, py_lint_op, py_test_op])
