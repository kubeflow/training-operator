# A very simple parameter server that joins the server defined by the cluster spec passed as environment variable

import tensorflow as tf
import os
import json

tf_config_json = os.environ.get("TF_CONFIG", "{}")
tf_config = json.loads(tf_config_json)
task = tf_config.get("task", {})
cluster_spec = tf_config.get("cluster", {})
cluster_spec_object = tf.train.ClusterSpec(cluster_spec)
job_name = task["type"]
task_id = task["index"]
server_def = tf.train.ServerDef(
		cluster=cluster_spec_object.as_cluster_def(),
		protocol="grpc",
		job_name=job_name,
		task_index=task_id)
server = tf.train.Server(server_def)

if job_name == 'ps':
    server.join()
