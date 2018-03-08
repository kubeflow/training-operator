import json
import unittest

import mock
from google.cloud import storage  # pylint: disable=no-name-in-module

from py import prow


class TestProw(unittest.TestCase):

  @mock.patch("prow.time.time")
  def testCreateFinished(self, mock_time):  # pylint: disable=no-self-use
    """Test create finished"""
    mock_time.return_value = 1000
    gcs_client = mock.MagicMock(spec=storage.Client)
    blob = prow.create_finished(gcs_client, "gs://bucket/output", True)

    expected = {
      "timestamp": 1000,
      "result": "SUCCESS",
      "metadata": {},
    }
    blob.upload_from_string.assert_called_once_with(json.dumps(expected))

  @mock.patch("prow.time.time")
  def testCreateStartedPeriodic(self, mock_time):  # pylint: disable=no-self-use
    """Test create started for periodic job."""
    mock_time.return_value = 1000
    gcs_client = mock.MagicMock(spec=storage.Client)
    blob = prow.create_started(gcs_client, "gs://bucket/output", "abcd")

    expected = {
      "timestamp": 1000,
      "repos": {
        "kubeflow/tf-operator": "abcd",
      },
    }
    blob.upload_from_string.assert_called_once_with(json.dumps(expected))

  def testGetSymlinkOutput(self):
    location = prow.get_symlink_output("10", "mlkube-build-presubmit", "20")
    self.assertEqual(
      "gs://kubernetes-jenkins/pr-logs/directory/mlkube-build-presubmit/20.txt",
      location)

  def testCreateSymlinkOutput(self):  # pylint: disable=no-self-use
    """Test create started for periodic job."""
    gcs_client = mock.MagicMock(spec=storage.Client)
    blob = prow.create_symlink(gcs_client, "gs://bucket/symlink",
                               "gs://bucket/output")

    blob.upload_from_string.assert_called_once_with("gs://bucket/output")

  @mock.patch("py.prow.test_util.get_num_failures")
  @mock.patch("py.prow._get_actual_junit_files")
  def testCheckNoErrorsSuccess(self, mock_get_junit, mock_get_failures):
    # Verify that check no errors returns true when there are no errors
    gcs_client = mock.MagicMock(spec=storage.Client)
    artifacts_dir = "gs://some_dir"
    mock_get_junit.return_value = set(["junit_1.xml"])
    mock_get_failures.return_value = 0
    junit_files = ["junit_1.xml"]
    self.assertTrue(
      prow.check_no_errors(gcs_client, artifacts_dir, junit_files))

  @mock.patch("py.prow.test_util.get_num_failures")
  @mock.patch("py.prow._get_actual_junit_files")
  def testCheckNoErrorsFailure(self, mock_get_junit, mock_get_failures):
    # Verify that check no errors returns false when a junit
    # file reports an error.
    gcs_client = mock.MagicMock(spec=storage.Client)
    artifacts_dir = "gs://some_dir"
    mock_get_junit.return_value = set(["junit_1.xml"])
    mock_get_failures.return_value = 1
    junit_files = ["junit_1.xml"]
    self.assertFalse(
      prow.check_no_errors(gcs_client, artifacts_dir, junit_files))

  @mock.patch("py.prow.test_util.get_num_failures")
  @mock.patch("py.prow._get_actual_junit_files")
  def testCheckNoErrorsFailureExtraJunit(self, mock_get_junit,
                                         mock_get_failures):
    # Verify that check no errors returns false when there are extra
    # junit files
    gcs_client = mock.MagicMock(spec=storage.Client)
    artifacts_dir = "gs://some_dir"
    mock_get_junit.return_value = set(["junit_0.xml", "junit_1.xml"])
    mock_get_failures.return_value = 0
    junit_files = ["junit_1.xml"]
    self.assertFalse(
      prow.check_no_errors(gcs_client, artifacts_dir, junit_files))


if __name__ == "__main__":
  unittest.main()
