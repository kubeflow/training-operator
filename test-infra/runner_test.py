import json
import unittest

import mock
from google.cloud import storage  # pylint: disable=no-name-in-module

import runner


class TestRunner(unittest.TestCase):
  @mock.patch("runner.time.time")
  def testCreateFinished(self, mock_time):  # pylint: disable=no-self-use
    """Test create finished"""
    mock_time.return_value = 1000
    gcs_client = mock.MagicMock(spec=storage.Client)
    blob = runner.create_finished(gcs_client, "gs://bucket/output", True)

    expected = {
        "timestamp": 1000,
        "result": "SUCCESS",
        "metadata": {},
    }
    blob.upload_from_string.assert_called_once_with(json.dumps(expected))

  @mock.patch("runner.time.time")
  def testCreateStartedPeriodic(self, mock_time):  # pylint: disable=no-self-use
    """Test create started for periodic job."""
    mock_time.return_value = 1000
    gcs_client = mock.MagicMock(spec=storage.Client)
    blob = runner.create_started(gcs_client, "gs://bucket/output", "abcd")

    expected = {
        "timestamp": 1000,
        "repos": {
            "jlewi/mlkube.io": "abcd",
        },
    }
    blob.upload_from_string.assert_called_once_with(json.dumps(expected))

  def testGetSymlinkOutput(self):
    location = runner.get_symlink_output("10", "mlkube-build-presubmit", "20")
    self.assertEquals(
        "gs://kubernetes-jenkins/pr-logs/directory/mlkube-build-presubmit/20.txt",
        location)

  def testCreateSymlinkOutput(self):  # pylint: disable=no-self-use
    """Test create started for periodic job."""
    gcs_client = mock.MagicMock(spec=storage.Client)
    blob = runner.create_symlink(gcs_client, "gs://bucket/symlink",
                                 "gs://bucket/output")

    blob.upload_from_string.assert_called_once_with("gs://bucket/output")


if __name__ == "__main__":
  unittest.main()
