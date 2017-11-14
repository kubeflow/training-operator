import google.auth
import google.auth.transport
import google.auth.transport.requests

import kubernetes
from kubernetes import client as k8s_client
from kubernetes import config as k8s_config
from kubernetes.config import kube_config
from kubernetes.client import ApiClient, ConfigurationObject, configuration
import os
import yaml
from py import util

util.configure_kubectl("mlkube-testing", "us-east1-d", "e2e-1114-022")
util.load_kube_config()
# Create an API client object to talk to the K8s master.
#api_client = k8s_client.ApiClient(config=k8s_client.configuration)


#k8s_config.load_kube_config()

api_client = k8s_client.ApiClient()

v1 = k8s_client.CoreV1Api(api_client)
print("Listing pods with their IPs:")
ret = v1.list_node(watch=False)
for i in ret.items:
    print("{0}".format(i))