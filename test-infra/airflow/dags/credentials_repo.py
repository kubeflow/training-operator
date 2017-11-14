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

def load_kube_config(config_file=None, context=None,
                     client_configuration=configuration,
                     persist_config=True, **kwargs):
  """Loads authentication and cluster information from kube-config file
  and stores them in kubernetes.client.configuration.

  :param config_file: Name of the kube-config file.
  :param context: set the active context. If is set to None, current_context
      from config file will be used.
  :param client_configuration: The kubernetes.client.ConfigurationObject to
      set configs to.
  :param persist_config: If True, config file will be updated when changed
      (e.g GCP token refresh).
  """

  if config_file is None:
    config_file = os.path.expanduser(kube_config.KUBE_CONFIG_DEFAULT_LOCATION)

  config_persister = None
  if persist_config:
    def _save_kube_config(config_map):
      with open(config_file, 'w') as f:
        yaml.safe_dump(config_map, f, default_flow_style=False)
    config_persister = _save_kube_config

  kube_config._get_kube_config_loader_for_yaml_file(
      config_file, active_context=context,
        client_configuration=client_configuration,
        config_persister=config_persister, **kwargs).load_and_set()

def refresh_credentials():
  #credentials, project_id = google.auth.default(scopes=["https://www.googleapis.com/auth/cloud-platform"])
  credentials, project_id = google.auth.default(scopes=["https://www.googleapis.com/auth/userinfo.email"])
  request = google.auth.transport.requests.Request()
  credentials.refresh(request)
  return credentials

refresh_credentials()
load_kube_config(get_google_credentials=refresh_credentials)
# Create an API client object to talk to the K8s master.
api_client = k8s_client.ApiClient()
