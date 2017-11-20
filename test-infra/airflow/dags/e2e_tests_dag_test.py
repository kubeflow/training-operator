"""Unittest for airflow."""
import tempfile
import unittest

import mock
import yaml

import e2e_tests_dag

class FakeTi(object):
  def __init__(self):
    self.topull = {}
    self.pushed = {}

  def xcom_pull(self, task_id, key=None):
    return self.topull.get(task_id, {}).get(key)

  def xcom_push(self, key=None, value=None):
    self.pushed[key] = value

  def set_pull(self, task_id, key=None, value=None):
    if not task_id in self.topull:
      self.topull[task_id] = {}
    self.topull[task_id][key] = value

class E22DagTest(unittest.TestCase):
  @mock.patch("e2e_tests_dag.util.run")
  def test_setup_cluster(self, mock_run):
    """Test setup cluster."""
    dag_run = e2e_tests_dag.FakeDagrun()
    dag_run.conf["ARTIFACTS_PATH"] = "gs://some/path"

    ti = FakeTi()
    ti.set_pull("build_images", "helm_chart", "gs://some/chart.tgz")
    #mock_trigger.return_value = ("2017-11-01T12:12:12", "")
    e2e_tests_dag.setup_cluster(dag_run, ti)

if __name__ == "__main__":
  unittest.main()
