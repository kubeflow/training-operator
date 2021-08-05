import json
import logging
import re
import subprocess

from kubeflow.testing import ks_util, test_util, util
from kubeflow.tf_operator import test_runner, tf_job_client
from kubeflow.tf_operator import util as tf_operator_util
from kubernetes import client as k8s_client

INVALID_TFJOB_COMPONENT_NAME = "invalid_tfjob"


class InvalidTfJobTests(test_util.TestCase):

  def __init__(self, args):
    namespace, name, env = test_runner.parse_runtime_params(args)
    self.app_dir = args.app_dir
    self.env = env
    self.namespace = namespace
    self.tfjob_version = args.tfjob_version
    self.params = args.params
    super(InvalidTfJobTests, self).__init__(
      class_name="InvalidTfJobTests", name=name)

  def test_invalid_tfjob_spec(self):
    tf_operator_util.load_kube_config()
    api_client = k8s_client.ApiClient()
    component = INVALID_TFJOB_COMPONENT_NAME + "_" + self.tfjob_version

    # Setup the ksonnet app
    tf_operator_util.setup_ks_app(self.app_dir, self.env, self.namespace, component,
                                  self.params)

    # Create the TF job
    ks_cmd = ks_util.get_ksonnet_cmd(self.app_dir)
    try:
      util.run([ks_cmd, "apply", self.env, "-c", component], cwd=self.app_dir)
    except subprocess.CalledProcessError as e:
      if "invalid: spec.tfReplicaSpecs: Required value" in e.output:
        logging.info("Created job failed which is expected. Reason %s", e.output)
      else:
        self.failure = "Job {0} in namespace {1} failed because {2}".format(self.name, self.namespace, e.output)
        logging.error(self.failure)


if __name__ == "__main__":
  test_runner.main(module=__name__)
