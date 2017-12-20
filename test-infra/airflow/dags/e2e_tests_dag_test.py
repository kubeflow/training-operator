"""Unittest for airflow."""
import unittest

import mock

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
  def test_pycommand(self, mock_run):  # pylint: disable=no-self-use
    """Test setup cluster."""
    py_op = e2e_tests_dag.py_checks_gen("lint")

    dag_run = e2e_tests_dag.FakeDagrun()
    dag_run.conf["ARTIFACTS_PATH"] = "gs://some/path"

    ti = FakeTi()
    ti.set_pull(None, "src_dir", "/some/dir")

    py_op(dag_run, ti)
    mock_run.assert_called_once_with(
      ['python', '-m', 'py.py_checks', 'lint', '--src_dir=/some/dir',
       '--junit_path=gs://some/path/junit_pycheckslint.xml',
       '--project=mlkube-testing'], dryrun=False, use_print=True)

  @mock.patch("e2e_tests_dag.util.run")
  def test_setup_cluster(self, _mock_run):  # pylint: disable=no-self-use
    """Test setup cluster."""
    dag_run = e2e_tests_dag.FakeDagrun()
    dag_run.conf["ARTIFACTS_PATH"] = "gs://some/path"

    ti = FakeTi()
    ti.set_pull("build_images", "helm_chart", "gs://some/chart.tgz")
    e2e_tests_dag.setup_cluster(dag_run, ti)

if __name__ == "__main__":
  unittest.main()
