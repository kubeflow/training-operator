from __future__ import print_function

from kubernetes import client as k8s_client
import mock
import unittest

from py import util

DEPLOYMENT_READY = """{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "extensions/v1beta1",
            "kind": "Deployment",
            "metadata": {
                "annotations": {
                    "deployment.kubernetes.io/revision": "1"
                },
                "creationTimestamp": "2017-12-22T13:58:29Z",
                "generation": 1,
                "labels": {
                    "name": "tf-job-operator"
                },
                "name": "tf-job-operator",
                "namespace": "default",
                "resourceVersion": "666",
                "selfLink": "/apis/extensions/v1beta1/namespaces/default/deployments/tf-job-operator",
                "uid": "3060361a-e720-11e7-ada7-42010a8e00a9"
            },
            "spec": {
                "replicas": 1,
                "selector": {
                    "matchLabels": {
                        "name": "tf-job-operator"
                    }
                },
                "strategy": {
                    "rollingUpdate": {
                        "maxSurge": 1,
                        "maxUnavailable": 1
                    },
                    "type": "RollingUpdate"
                },
                "template": {
                    "metadata": {
                        "creationTimestamp": null,
                        "labels": {
                            "name": "tf-job-operator"
                        }
                    },
                    "spec": {
                        "containers": [
                            {
                                "command": [
                                    "/opt/mlkube/tf_operator",
                                    "--controller_config_file=/etc/config/controller_config_file.yaml",
                                    "-alsologtostderr",
                                    "-v=1"
                                ],
                                "env": [
                                    {
                                        "name": "MY_POD_NAMESPACE",
                                        "valueFrom": {
                                            "fieldRef": {
                                                "apiVersion": "v1",
                                                "fieldPath": "metadata.namespace"
                                            }
                                        }
                                    },
                                    {
                                        "name": "MY_POD_NAME",
                                        "valueFrom": {
                                            "fieldRef": {
                                                "apiVersion": "v1",
                                                "fieldPath": "metadata.name"
                                            }
                                        }
                                    }
                                ],
                                "image": "gcr.io/mlkube-testing/tf_operator:v20171222-e108d55",
                                "imagePullPolicy": "IfNotPresent",
                                "name": "tf-job-operator",
                                "resources": {},
                                "terminationMessagePath": "/dev/termination-log",
                                "terminationMessagePolicy": "File",
                                "volumeMounts": [
                                    {
                                        "mountPath": "/etc/config",
                                        "name": "config-volume"
                                    }
                                ]
                            }
                        ],
                        "dnsPolicy": "ClusterFirst",
                        "restartPolicy": "Always",
                        "schedulerName": "default-scheduler",
                        "securityContext": {},
                        "serviceAccount": "tf-job-operator",
                        "serviceAccountName": "tf-job-operator",
                        "terminationGracePeriodSeconds": 30,
                        "volumes": [
                            {
                                "configMap": {
                                    "defaultMode": 420,
                                    "name": "tf-job-operator-config"
                                },
                                "name": "config-volume"
                            }
                        ]
                    }
                }
            },
            "status": {
                "availableReplicas": 1,
                "conditions": [
                    {
                        "lastTransitionTime": "2017-12-22T13:58:29Z",
                        "lastUpdateTime": "2017-12-22T13:58:29Z",
                        "message": "Deployment has minimum availability.",
                        "reason": "MinimumReplicasAvailable",
                        "status": "True",
                        "type": "Available"
                    }
                ],
                "observedGeneration": 1,
                "readyReplicas": 1,
                "replicas": 1,
                "updatedReplicas": 1
            }
        }
    ],
    "kind": "List",
    "metadata": {
        "resourceVersion": "",
        "selfLink": ""
    }
}"""

class UtilTest(unittest.TestCase):
  def test_wait_for_deployment(self):
    api_client = mock.MagicMock(spec=k8s_client.ApiClient)

    response = k8s_client.ExtensionsV1beta1Deployment()
    response.status = k8s_client.ExtensionsV1beta1DeploymentStatus()
    response.status.ready_replicas = 1
    api_client.call_api.return_value = response
    util.wait_for_deployment(api_client, "some-namespace", "some-deployment")


if __name__ == "__main__":
  unittest.main()
