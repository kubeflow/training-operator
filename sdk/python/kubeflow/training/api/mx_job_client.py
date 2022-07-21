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
import threading
import queue

from kubernetes import client, config
from kubernetes import watch as k8s_watch

from kubeflow.training.constants import constants
from kubeflow.training.utils import utils

from .mx_job_watch import watch as mxjob_watch

logging.basicConfig(format='%(message)s')
logging.getLogger().setLevel(logging.INFO)


def wrap_log_stream(q, stream):
    while True:
        try:
            logline = next(stream)
            q.put(logline)
        except StopIteration:
            q.put(None)
            return
        except Exception as e:
            raise RuntimeError(
                "Exception when calling CoreV1Api->read_namespaced_pod_log: %s\n" % e)


def get_log_queue_pool(streams):
    pool = []
    for stream in streams:
        q = queue.Queue(maxsize=100)
        pool.append(q)
        threading.Thread(target=wrap_log_stream, args=(q, stream)).start()
    return pool


class MXJobClient(object):
    def __init__(self, config_file=None, context=None,  # pylint: disable=too-many-arguments
                 client_configuration=None, persist_config=True):
        """
        MXJob client constructor
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

    def create(self, mxjob, namespace=None):
        """
        Create the MXJob
        :param mxjob: mxjob object
        :param namespace: defaults to current or default namespace
        :return: created mxjob
        """

        if namespace is None:
            namespace = utils.set_mxjob_namespace(mxjob)

        try:
            outputs = self.custom_api.create_namespaced_custom_object(
                constants.MXJOB_GROUP,
                constants.MXJOB_VERSION,
                namespace,
                constants.MXJOB_PLURAL,
                mxjob)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->create_namespaced_custom_object:\
                 %s\n" % e)

        return outputs

    def get(self, name=None, namespace=None, watch=False,
            timeout_seconds=600):  # pylint: disable=inconsistent-return-statements
        """
        Get the mxjob
        :param name: existing mxjob name, if not defined, the get all mxjobs in the namespace.
        :param namespace: defaults to current or default namespace
        :param watch: Watch the MXJob if `True`.
        :param timeout_seconds: How long to watch the job..
        :return: mxjob
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        if name:
            if watch:
                mxjob_watch(
                    name=name,
                    namespace=namespace,
                    timeout_seconds=timeout_seconds)
            else:
                thread = self.custom_api.get_namespaced_custom_object(
                    constants.MXJOB_GROUP,
                    constants.MXJOB_VERSION,
                    namespace,
                    constants.MXJOB_PLURAL,
                    name,
                    async_req=True)

                mxjob = None
                try:
                    mxjob = thread.get(constants.APISERVER_TIMEOUT)
                except multiprocessing.TimeoutError:
                    raise RuntimeError("Timeout trying to get MXJob.")
                except client.rest.ApiException as e:
                    raise RuntimeError(
                        "Exception when calling CustomObjectsApi->get_namespaced_custom_object:\
                        %s\n" % e)
                except Exception as e:
                    raise RuntimeError(
                        "There was a problem to get MXJob {0} in namespace {1}. Exception: \
                        {2} ".format(name, namespace, e))
                return mxjob
        else:
            if watch:
                mxjob_watch(
                    namespace=namespace,
                    timeout_seconds=timeout_seconds)
            else:
                thread = self.custom_api.list_namespaced_custom_object(
                    constants.MXJOB_GROUP,
                    constants.MXJOB_VERSION,
                    namespace,
                    constants.MXJOB_PLURAL,
                    async_req=True)

                mxjobs = None
                try:
                    mxjobs = thread.get(constants.APISERVER_TIMEOUT)
                except multiprocessing.TimeoutError:
                    raise RuntimeError("Timeout trying to get MXJob.")
                except client.rest.ApiException as e:
                    raise RuntimeError(
                        "Exception when calling CustomObjectsApi->list_namespaced_custom_object:\
                        %s\n" % e)
                except Exception as e:
                    raise RuntimeError(
                        "There was a problem to list MXJobs in namespace {0}. \
                        Exception: {1} ".format(namespace, e))
                return mxjobs

    def patch(self, name, mxjob, namespace=None):
        """
        Patch existing mxjob
        :param name: existing mxjob name
        :param mxjob: patched mxjob
        :param namespace: defaults to current or default namespace
        :return: patched mxjob
        """
        if namespace is None:
            namespace = utils.set_mxjob_namespace(mxjob)

        try:
            outputs = self.custom_api.patch_namespaced_custom_object(
                constants.MXJOB_GROUP,
                constants.MXJOB_VERSION,
                namespace,
                constants.MXJOB_PLURAL,
                name,
                mxjob)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->patch_namespaced_custom_object:\
                 %s\n" % e)

        return outputs

    def delete(self, name, namespace=None):
        """
        Delete the mxjob
        :param name: mxjob name
        :param namespace: defaults to current or default namespace
        :return:
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        try:
            return self.custom_api.delete_namespaced_custom_object(
                group=constants.MXJOB_GROUP,
                version=constants.MXJOB_VERSION,
                namespace=namespace,
                plural=constants.MXJOB_PLURAL,
                name=name,
                body=client.V1DeleteOptions())
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->delete_namespaced_custom_object:\
                 %s\n" % e)

    def wait_for_job(self, name,  # pylint: disable=inconsistent-return-statements
                     namespace=None,
                     timeout_seconds=600,
                     polling_interval=30,
                     watch=False,
                     status_callback=None):
        """Wait for the specified job to finish.

        :param name: Name of the TfJob.
        :param namespace: defaults to current or default namespace.
        :param timeout_seconds: How long to wait for the job.
        :param polling_interval: How often to poll for the status of the job.
        :param watch: Watch the MXJob if `True`.
        :param status_callback: (Optional): Callable. If supplied this callable is
               invoked after we poll the job. Callable takes a single argument which
               is the job.
        :return:
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        if watch:
            mxjob_watch(
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
        :return: Object MXJob status
        """

        if namespace is None:
            namespace = utils.get_default_target_namespace()

        for _ in range(round(timeout_seconds / polling_interval)):

            mxjob = None
            mxjob = self.get(name, namespace=namespace)

            if mxjob:
                if status_callback:
                    status_callback(mxjob)

                # If we poll the CRD quick enough status won't have been set yet.
                conditions = mxjob.get("status", {}).get("conditions", [])
                # Conditions might have a value of None in status.
                conditions = conditions or []
                for c in conditions:
                    if c.get("type", "") in expected_condition:
                        return mxjob

            time.sleep(polling_interval)

        raise RuntimeError(
            "Timeout waiting for MXJob {0} in namespace {1} to enter one of the "
            "conditions {2}.".format(name, namespace, expected_condition), mxjob)

    def get_job_status(self, name, namespace=None):
        """Returns MXJob status, such as Running, Failed or Succeeded.

        :param name: The MXJob name.
        :param namespace: defaults to current or default namespace.
        :return: Object MXJob status
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        mxjob = self.get(name, namespace=namespace)
        last_condition = mxjob.get("status", {}).get("conditions", [{}])[-1]
        return last_condition.get("type", "")

    def is_job_running(self, name, namespace=None):
        """Returns true if the MXJob running; false otherwise.

        :param name: The MXJob name.
        :param namespace: defaults to current or default namespace.
        :return: True or False
        """
        mxjob_status = self.get_job_status(name, namespace=namespace)
        return mxjob_status.lower() == "running"

    def is_job_succeeded(self, name, namespace=None):
        """Returns true if the MXJob succeeded; false otherwise.

        :param name: The MXJob name.
        :param namespace: defaults to current or default namespace.
        :return: True or False
        """
        mxjob_status = self.get_job_status(name, namespace=namespace)
        return mxjob_status.lower() == "succeeded"

    def get_pod_names(self, name, namespace=None, master=False,  # pylint: disable=inconsistent-return-statements
                      replica_type=None, replica_index=None):
        """
        Get pod names of MXJob.
        :param name: mxjob name
        :param namespace: defaults to current or default namespace.
        :param master: Only get pod with label 'job-role: master' pod if True.
        :param replica_type: User can specify one of 'worker, ps, chief' to only get one type pods.
               By default get all type pods.
        :param replica_index: User can specfy replica index to get one pod of MXJob.
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
            logging.warning("Not found Pods of the MXJob %s with the labels %s.", name, labels)
        else:
            return set(pod_names)

    def get_logs(self, name, namespace=None, master=True,
                 replica_type=None, replica_index=None,
                 follow=False, container="mxnet"):
        """
        Get training logs of the MXJob.
        By default only get the logs of Pod that has labels 'job-role: master'.
        :param container: container name
        :param name: mxjob name
        :param namespace: defaults to current or default namespace.
        :param master: By default get pod with label 'job-role: master' pod if True.
                       If need to get more Pod Logs, set False.
        :param replica_type: User can specify one of 'worker, ps, chief' to only get one type pods.
               By default get all type pods.
        :param replica_index: User can specfy replica index to get one pod of MXJob.
        :param follow: Follow the log stream of the pod. Defaults to false.
        :return: str: pods logs
        """

        if namespace is None:
            namespace = utils.get_default_target_namespace()

        pod_names = list(self.get_pod_names(name, namespace=namespace,
                                            master=master,
                                            replica_type=replica_type,
                                            replica_index=replica_index))
        if pod_names:
            if follow:
                log_streams = []
                for pod in pod_names:
                    log_streams.append(k8s_watch.Watch().stream(self.core_api.read_namespaced_pod_log,
                                                                name=pod, namespace=namespace, container=container))
                finished = [False for _ in log_streams]

                # create thread and queue per stream, for non-blocking iteration
                log_queue_pool = get_log_queue_pool(log_streams)

                # iterate over every watching pods' log queue
                while True:
                    for index, log_queue in enumerate(log_queue_pool):
                        if all(finished):
                            return
                        if finished[index]:
                            continue
                        # grouping the every 50 log lines of the same pod
                        for _ in range(50):
                            try:
                                logline = log_queue.get(timeout=1)
                                if logline is None:
                                    finished[index] = True
                                    break
                                logging.info("[Pod %s]: %s", pod_names[index], logline)
                            except queue.Empty:
                                break
            else:
                for pod in pod_names:
                    try:
                        pod_logs = self.core_api.read_namespaced_pod_log(pod, namespace, container=container)
                        logging.info("The logs of Pod %s:\n %s", pod, pod_logs)
                    except client.rest.ApiException as e:
                        raise RuntimeError(
                            "Exception when calling CoreV1Api->read_namespaced_pod_log: %s\n" % e)
        else:
            raise RuntimeError("Not found Pods of the MXJob {} "
                               "in namespace {}".format(name, namespace))
