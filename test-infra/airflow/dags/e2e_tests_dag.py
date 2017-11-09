"""
An Airflow pipeline for running our E2E tests.
"""

from airflow import DAG
from airflow.operators import PythonOperator
from datetime import datetime, timedelta
import logging
import os
import tempfile
from py import release
from py import util

default_args = {
  'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime(2015, 6, 1),
    'email': ['airflow@airflow.com'],
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
    # 'queue': 'bash_queue',
    # 'pool': 'backfill',
    # 'priority_weight': 10,
    # 'end_date': datetime(2016, 1, 1),
}

dag = DAG(
  # Set schedule_interval to None
    'tf_k8s_tests', default_args=default_args, schedule_interval=None)

# Default name for the repo organization and name.
# This should match the values used in Go imports.
DEFAULT_REPO_OWNER = "tensorflow"
DEFAULT_REPO_NAME = "k8s"

# TODO(jlewi): Can we eliminate code duplication with bootstrap.py? Although
# bootstrap.py should potentially not need to clone repo once we update it
# to just trigger Airflow jobs.
# TODO(jlewi): Can we merge with util.clone_repo?
def clone_repo(work_dir, repo_owner, repo_name, pull_number="",
               pull_pull_sha="", pull_base_sha=""):
  """Clone the repo.

  Args:
    work_dir: The directory to use as the working directory.

  Returns:
    src_path: This is the root path for the training code.
    sha: The sha number of the repo.
  """
  # Need to import it here so it works when actually invoked by Airflow.
  import shutil
  go_path = os.path.join(work_dir, "go")

  if bool(repo_owner) != bool(repo_name):
    raise ValueError("Either set both environment variables "
                         "REPO_OWNER and REPO_NAME or set neither.")

  src_dir = os.path.join(go_path, "src/github.com", DEFAULT_REPO_OWNER)
  dest = os.path.join(src_dir, DEFAULT_REPO_NAME)

  # Clone the repo
  repo = "https://github.com/{0}/{1}.git".format(repo_owner, repo_name)
  os.makedirs(src_dir)

  # TODO(jlewi): How can we figure out what branch
  util.run(["git", "clone", repo, dest])

  # If this is a presubmit PULL_PULL_SHA will be set see:
  # https://github.com/kubernetes/test-infra/tree/master/prow#job-evironment-variables
  sha = ""

  if pull_number:
    sha = pull_pull_sha
    # Its a presubmit job sob since we are testing a Pull request.
    run(["git", "fetch", "origin",
         "pull/{0}/head:pr".format(pull_number)],
         cwd=dest)
  else:
    # For postsubmits PULL_BASE_SHA will be set
    sha = pull_base_sha

  if sha:
    util.run(["git", "checkout", sha], cwd=dest)

  # Get the actual git hash.
  # This ensures even for periodic jobs which don't set the sha we know
  # the version of the code tested.
  sha = util.run_and_output(["git", "rev-parse", "HEAD"], cwd=dest)

  # Install dependencies
  # TODO(jlewi): We should probably move this into runner.py so that
  # output is captured in build-log.txt
  env = os.environ.copy()
  env["GOPATH"] = go_path
  util.run(["glide", "install"], cwd=dest, env=env)

  # We need to remove the vendored dependencies of apiextensions otherwise we
  # get type conflicts.
  shutil.rmtree(os.path.join(dest,
                             "vendor/k8s.io/apiextensions-apiserver/vendor"))

  return dest, sha


def build_images(dag_run=None, **kwargs):
  """
  Args:
    dag_run: A DagRun object. This is passed in as a result of setting
      provide_context to true for the operator.
  """
  # Create a temporary directory suitable for checking out and building the
  # code.
  logging.info("dag_id: %s", dag_run.dag_id)
  logging.info("run_id: %s", dag_run.run_id)
  work_dir = os.path.join(tempfile.gettempdir(), dag_run.dag_id, dag_run.run_id)
  # : cause problems with go paths.
  work_dir = work_dir.replace(":", "_")
  logging.info("Using working directory: %s", work_dir)

  conf = dag_run.conf
  if not conf:
    conf = {}

  repo_owner = conf.get("REPO_OWNER", DEFAULT_REPO_OWNER)
  repo_name = conf.get("REPO_NAME", DEFAULT_REPO_NAME)

  pull_number = conf.get("PULL_NUMBER", "")
  pull_pull_sha = conf.get("PULL_PULL_SHA", "")
  pull_base_sha = conf.get("PULL_BASE_SHA", "")

  clone_repo(work_dir, repo_owner, repo_name, pull_number=pull_number,
             pull_pull_sha=pull_pull_sha, pull_base_sha=pull_base_sha)

  # Create a directory to use as the GOPATH
  go_path = os.path.join(work_dir, "go")
  go_src = os.path.join(go_path, "src")
  if not os.path.exists(go_src):
    os.makedirs(go_src)

def setup_cluster():
  print("setup cluster")

def create_cluster():
  print("Create cluster")

def deploy_crd():
  print("Deploy crd")

def run_tests():
  print("Run tests")

def delete_cluster():
  print("Delete cluster")


build_op = PythonOperator(
  task_id='build_images',
    provide_context=True,
    python_callable=build_images,
    dag=dag)

#create_cluster_op = PythonOperator(
  #task_id='create_cluster',
    #provide_context=False,
    #python_callable=create_cluster,
    #dag=dag)

#setup_cluster_op = PythonOperator(
  #task_id='setup_cluster',
    #provide_context=False,
    #python_callable=setup_cluster,
    #dag=dag)

#setup_cluster_op.set_upstream(create_cluster_op)


#deploy_crd_op = PythonOperator(
  #task_id='deploy_crd',
    #provide_context=False,
    #python_callable=deploy_crd,
    #dag=dag)

#deploy_crd_op.set_upstream([setup_cluster_op, build_op])

#run_tests_op = PythonOperator(
  #task_id='run_tests',
    #provide_context=False,
    #python_callable=run_tests,
    #dag=dag)

#run_tests_op.set_upstream(deploy_crd_op)

#delete_cluster_op = PythonOperator(
  #task_id='delete_cluster',
    #provide_context=False,
    #python_callable=run_tests,
    #dag=dag)


#delete_cluster_op.set_upstream(run_tests_op)
