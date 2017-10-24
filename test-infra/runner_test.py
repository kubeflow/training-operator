import json
import mock
import runner
import unittest

from google.cloud import storage

class TestRunner(unittest.TestCase):
  @mock.patch("runner.time.time")
  def testCreateStartedPeriodic(self, mock_time):
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


if __name__ == "__main__":
  unittest.main()
