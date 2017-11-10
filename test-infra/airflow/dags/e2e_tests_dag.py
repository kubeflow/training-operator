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

# GCS path to use for each dag run
GCS_RUNS_PATH = "gs://mlkube-testing-airflow/runs"

GCB_PROJECT = "mlkube-testing"

def run_path(dag_id, run_id):
  return os.path.join(GCS_RUNS_PATH, dag_id.replace(":", "_"),
                      run_id.replace(":", "_"))

class FakeDagrun(object):
  def __init__(self):
    self.dag_id = "tf_k8s_tests"
    self.run_id = "test_run"
    self.conf = {}

def build_images(dag_run=None, **kwargs):
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

  gcs_path = run_path(dag_run.dag_id, dag_run.run_id)
  logging.info("gcs_path %s", gcs_path)

  work_dir = os.path.join(tempfile.gettempdir(), dag_run.dag_id, dag_run.run_id)

  conf = dag_run.conf
  if not conf:
    conf = {}

  repo_owner = conf.get("REPO_OWNER", DEFAULT_REPO_OWNER)
  repo_name = conf.get("REPO_NAME", DEFAULT_REPO_NAME)

  pull_number = conf.get("PULL_NUMBER", "")
  args = ["python", "-m", "py.release"]
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

  args.append("--build_info_path=" + os.path.join(gcs_path, "build_info.yaml"))
  args.append("--releases_path=" + os.path.join(gcs_path))
  args.append("--project=" + GCB_PROJECT)
  # We want subprocess output to bypass logging module otherwise multiline
  # output is squashed together.
  util.run(args, use_print=True)

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
