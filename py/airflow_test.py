"""Unittest for airflow."""
import datetime
import unittest

import mock

from py import airflow


class AirflowTest(unittest.TestCase):
  @mock.patch('py.airflow.prow')
  @mock.patch('py.airflow.wait_for_tf_k8s_tests')
  @mock.patch('py.airflow.trigger_tf_k8s_tests_dag')
  def test_main(self, mock_trigger, mock_wait, _mock_prow):  # pylint: disable=no-self-use
    mock_trigger.return_value = ("2017-11-01T12:12:12", "")
    mock_wait.return_value = "success"
    airflow.main()

  def test_wait_for_tf_k8s_tests(self):  # pylint: disable=no-self-use
    client = mock.MagicMock(spec=airflow.AirflowClient)
    client.get_task_status.side_effect = [
        {"state": None},
        {"state": "running"},
        {"state": "success"},
      ]

    state = airflow.wait_for_tf_k8s_tests(client, "2017-11-01T12:12:12",
                                          polling_interval=datetime.timedelta(seconds=0))

    self.assertEquals("success", state)
if __name__ == "__main__":
  unittest.main()
