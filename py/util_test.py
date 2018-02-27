from __future__ import print_function

import unittest

from kubernetes import client as k8s_client
import mock

from py import util


class UtilTest(unittest.TestCase):

  def test_wait_for_deployment(self):
    api_client = mock.MagicMock(spec=k8s_client.ApiClient)

    response = k8s_client.ExtensionsV1beta1Deployment()
    response.status = k8s_client.ExtensionsV1beta1DeploymentStatus()
    response.status.ready_replicas = 1
    api_client.call_api.return_value = response
    result = util.wait_for_deployment(api_client, "some-namespace",
                                      "some-deployment")
    self.assertIsNotNone(result)

  def test_wait_for_statefulset(self):
    api_client = mock.MagicMock(spec=k8s_client.ApiClient)

    response = k8s_client.V1beta1StatefulSet()
    response.status = k8s_client.V1beta1StatefulSetStatus(
      ready_replicas=1, replicas=1)
    api_client.call_api.return_value = response
    result = util.wait_for_statefulset(api_client, "some-namespace", "some-set")
    self.assertIsNotNone(result)

  def testSplitGcsUri(self):
    bucket, path = util.split_gcs_uri("gs://some-bucket/some/path")
    self.assertEqual("some-bucket", bucket)
    self.assertEqual("some/path", path)

    bucket, path = util.split_gcs_uri("gs://some-bucket")
    self.assertEqual("some-bucket", bucket)
    self.assertEqual("", path)


if __name__ == "__main__":
  unittest.main()
