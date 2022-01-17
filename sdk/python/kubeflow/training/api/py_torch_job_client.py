# Copyright 2021 The Kubeflow Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import multiprocessing
import time
import logging

from kubernetes import client, config

from kubeflow.training.constants import constants
from kubeflow.training.utils import utils

from .py_torch_job_watch import watch as pytorchjob_watch

logging.basicConfig(format='%(message)s')
logging.getLogger().setLevel(logging.INFO)


class PyTorchJobClient(object):
    def __init__(self, config_file=None, context=None,  # pylint: disable=too-many-arguments
                 client_configuration=None, persist_config=True):
        """
        PyTorchJob client constructor
        :param config_file: kubeconfig file, defaults to ~/.kube/config
        :param context: kubernetes context
        :param client_configuration: kubernetes configuration object
        :param persist_config:
        """
        if config_file or not utils.is_running_in_k8s():
            config.load_kube_config(
                config_file=config_file,
                context=context,
                client_configuration=client_configuration,
                persist_config=persist_config)
        else:
            config.load_incluster_config()

        self.custom_api = client.CustomObjectsApi()
        self.core_api = client.CoreV1Api()

    def create(self, pytorchjob, namespace=None):
        """
        Create the PyTorchJob
        :param pytorchjob: pytorchjob object
        :param namespace: defaults to current or default namespace
        :return: created pytorchjob
        """

        if namespace is None:
            namespace = utils.set_pytorchjob_namespace(pytorchjob)

        try:
            outputs = self.custom_api.create_namespaced_custom_object(
                constants.PYTORCHJOB_GROUP,
                constants.PYTORCHJOB_VERSION,
                namespace,
                constants.PYTORCHJOB_PLURAL,
                pytorchjob)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->create_namespaced_custom_object:\
                 %s\n" % e)

        return outputs

    def get(self, name=None, namespace=None, watch=False,
            timeout_seconds=600):  # pylint: disable=inconsistent-return-statements
        """
        Get the pytorchjob
        :param name: existing pytorchjob name, if not defined, get all pytorchjobs in the namespace.
        :param namespace: defaults to current or default namespace
        :param watch: Watch the pytorchjob if `True`.
        :param timeout_seconds: How long to watch the pytorchjob.
        :return: pytorchjob
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        if name:
            if watch:
                pytorchjob_watch(
                    name=name,
                    namespace=namespace,
                    timeout_seconds=timeout_seconds)
            else:
                thread = self.custom_api.get_namespaced_custom_object(
                    constants.PYTORCHJOB_GROUP,
                    constants.PYTORCHJOB_VERSION,
                    namespace,
                    constants.PYTORCHJOB_PLURAL,
                    name,
                    async_req=True)

                pytorchjob = None
                try:
                    pytorchjob = thread.get(constants.APISERVER_TIMEOUT)
                except multiprocessing.TimeoutError:
                    raise RuntimeError("Timeout trying to get PyTorchJob.")
                except client.rest.ApiException as e:
                    raise RuntimeError(
                        "Exception when calling CustomObjectsApi->get_namespaced_custom_object:\
                        %s\n" % e)
                except Exception as e:
                    raise RuntimeError(
                        "There was a problem to get PyTorchJob {0} in namespace {1}. Exception: \
                        {2} ".format(name, namespace, e))
                return pytorchjob
        else:
            if watch:
                pytorchjob_watch(
                    namespace=namespace,
                    timeout_seconds=timeout_seconds)
            else:
                thread = self.custom_api.list_namespaced_custom_object(
                    constants.PYTORCHJOB_GROUP,
                    constants.PYTORCHJOB_VERSION,
                    namespace,
                    constants.PYTORCHJOB_PLURAL,
                    async_req=True)

                pytorchjob = None
                try:
                    pytorchjob = thread.get(constants.APISERVER_TIMEOUT)
                except multiprocessing.TimeoutError:
                    raise RuntimeError("Timeout trying to get PyTorchJob.")
                except client.rest.ApiException as e:
                    raise RuntimeError(
                        "Exception when calling CustomObjectsApi->list_namespaced_custom_object: \
                        %s\n" % e)
                except Exception as e:
                    raise RuntimeError(
                        "There was a problem to List PyTorchJob in namespace {0}. \
                        Exception: {1} ".format(namespace, e))

                return pytorchjob

    def patch(self, name, pytorchjob, namespace=None):
        """
        Patch existing pytorchjob
        :param name: existing pytorchjob name
        :param pytorchjob: patched pytorchjob
        :param namespace: defaults to current or default namespace
        :return: patched pytorchjob
        """
        if namespace is None:
            namespace = utils.set_pytorchjob_namespace(pytorchjob)

        try:
            outputs = self.custom_api.patch_namespaced_custom_object(
                constants.PYTORCHJOB_GROUP,
                constants.PYTORCHJOB_VERSION,
                namespace,
                constants.PYTORCHJOB_PLURAL,
                name,
                pytorchjob)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->patch_namespaced_custom_object:\
                 %s\n" % e)

        return outputs

    def delete(self, name, namespace=None):
        """
        Delete the pytorchjob
        :param name: pytorchjob name
        :param namespace: defaults to current or default namespace
        :return:
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        try:
            return self.custom_api.delete_namespaced_custom_object(
                group=constants.PYTORCHJOB_GROUP,
                version=constants.PYTORCHJOB_VERSION,
                namespace=namespace,
                plural=constants.PYTORCHJOB_PLURAL,
                name=name,
                body=client.V1DeleteOptions())
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->delete_namespaced_custom_object:\
                 %s\n" % e)

    def wait_for_job(self, name,  # pylint: disable=inconsistent-return-statements
                     namespace=None,
                     watch=False,
                     timeout_seconds=600,
                     polling_interval=30,
                     status_callback=None):
        """Wait for the specified job to finish.

        :param name: Name of the PyTorchJob.
        :param namespace: defaults to current or default namespace.
        :param timeout_seconds: How long to wait for the job.
        :param polling_interval: How often to poll for the status of the job.
        :param status_callback: (Optional): Callable. If supplied this callable is
               invoked after we poll the job. Callable takes a single argument which
               is the job.
        :return:
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        if watch:
            pytorchjob_watch(
                name=name,
                namespace=namespace,
                timeout_seconds=timeout_seconds)
        else:
            return self.wait_for_condition(
                name,
                ["Succeeded", "Failed"],
                namespace=namespace,
                timeout_seconds=timeout_seconds,
                polling_interval=polling_interval,
                status_callback=status_callback)

    def wait_for_condition(self, name,
                           expected_condition,
                           namespace=None,
                           timeout_seconds=600,
                           polling_interval=30,
                           status_callback=None):
        """Waits until any of the specified conditions occur.

        :param name: Name of the job.
        :param expected_condition: A list of conditions. Function waits until any of the
               supplied conditions is reached.
        :param namespace: defaults to current or default namespace.
        :param timeout_seconds: How long to wait for the job.
        :param polling_interval: How often to poll for the status of the job.
        :param status_callback: (Optional): Callable. If supplied this callable is
               invoked after we poll the job. Callable takes a single argument which
               is the job.
        :return: Object: PyTorchJob
        """

        if namespace is None:
            namespace = utils.get_default_target_namespace()

        for _ in range(round(timeout_seconds / polling_interval)):

            pytorchjob = None
            pytorchjob = self.get(name, namespace=namespace)

            if pytorchjob:
                if status_callback:
                    status_callback(pytorchjob)

                # If we poll the CRD quick enough status won't have been set yet.
                conditions = pytorchjob.get("status", {}).get("conditions", [])
                # Conditions might have a value of None in status.
                conditions = conditions or []
                for c in conditions:
                    if c.get("type", "") in expected_condition:
                        return pytorchjob

            time.sleep(polling_interval)

        raise RuntimeError(
            "Timeout waiting for PyTorchJob {0} in namespace {1} to enter one of the "
            "conditions {2}.".format(name, namespace, expected_condition), pytorchjob)

    def get_job_status(self, name, namespace=None):
        """Returns PyTorchJob status, such as Running, Failed or Succeeded.

        :param name: The PyTorchJob name.
        :param namespace: defaults to current or default namespace.
        :return: str: PyTorchJob status
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        pytorchjob = self.get(name, namespace=namespace)
        last_condition = pytorchjob.get("status", {}).get("conditions", [])[-1]
        return last_condition.get("type", "")

    def is_job_running(self, name, namespace=None):
        """Returns true if the PyTorchJob running; false otherwise.

        :param name: The PyTorchJob name.
        :param namespace: defaults to current or default namespace.
        :return: True or False
        """
        pytorchjob_status = self.get_job_status(name, namespace=namespace)
        return pytorchjob_status == constants.JOB_STATUS_RUNNING

    def is_job_succeeded(self, name, namespace=None):
        """Returns true if the PyTorchJob succeeded; false otherwise.

        :param name: The PyTorchJob name.
        :param namespace: defaults to current or default namespace.
        :return: True or False
        """
        pytorchjob_status = self.get_job_status(name, namespace=namespace)
        return pytorchjob_status == constants.JOB_STATUS_SUCCEEDED

    def get_pod_names(self, name, namespace=None, master=False,  # pylint: disable=inconsistent-return-statements
                      replica_type=None, replica_index=None):
        """
        Get pod names of PyTorchJob.
        :param name: PyTorchJob name
        :param namespace: defaults to current or default namespace.
        :param master: Only get pod with label 'job-role: master' pod if True.
        :param replica_type: User can specify one of 'master, worker' to only get one type pods.
               By default get all type pods.
        :param replica_index: User can specfy replica index to get one pod of PyTorchJob.
        :return: set: pods name
        """

        if namespace is None:
            namespace = utils.get_default_target_namespace()

        labels = utils.get_job_labels(name, master=master,
                                             replica_type=replica_type,
                                             replica_index=replica_index)

        try:
            resp = self.core_api.list_namespaced_pod(
                namespace, label_selector=utils.to_selector(labels))
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CoreV1Api->read_namespaced_pod_log: %s\n" % e)

        pod_names = []
        for pod in resp.items:
            if pod.metadata and pod.metadata.name:
                pod_names.append(pod.metadata.name)

        if not pod_names:
            logging.warning("Not found Pods of the PyTorchJob %s with the labels %s.", name, labels)
        else:
            return set(pod_names)

    def get_logs(self, name, namespace=None, master=True,
                 replica_type=None, replica_index=None,
                 follow=False, container="pytorch"):
        """
        Get training logs of the PyTorchJob.
        By default only get the logs of Pod that has labels 'job-role: master'.
        :param container: container name
        :param name: PyTorchJob name
        :param namespace: defaults to current or default namespace.
        :param master: By default get pod with label 'job-role: master' pod if True.
                       If need to get more Pod Logs, set False.
        :param replica_type: User can specify one of 'master, worker' to only get one type pods.
               By default get all type pods.
        :param replica_index: User can specfy replica index to get one pod of PyTorchJob.
        :param follow: Follow the log stream of the pod. Defaults to false.
        :return: str: pods logs
        """

        if namespace is None:
            namespace = utils.get_default_target_namespace()

        pod_names = self.get_pod_names(name, namespace=namespace,
                                       master=master,
                                       replica_type=replica_type,
                                       replica_index=replica_index)

        if pod_names:
            for pod in pod_names:
                try:
                    pod_logs = self.core_api.read_namespaced_pod_log(
                        pod, namespace, follow=follow, container=container)
                    logging.info("The logs of Pod %s:\n %s", pod, pod_logs)
                except client.rest.ApiException as e:
                    raise RuntimeError(
                        "Exception when calling CoreV1Api->read_namespaced_pod_log: %s\n" % e)
        else:
            raise RuntimeError("Not found Pods of the PyTorchJob {} "
                               "in namespace {}".format(name, namespace))
