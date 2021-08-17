# Befor running this script, make sure tvm is install in your cluster

import os
import time
import json

if __name__ == '__main__':
    mx_config = json.loads(os.environ.get('MX_CONFIG') or '{}')
    cluster_config = mx_config.get('cluster', {})
    labels_config = mx_config.get('labels', {})
    task_config = mx_config.get('task', {})
    task_type = task_config.get('type')
    task_index = task_config.get('index')

    if task_type == "":
        print("No task_type, Error")
    elif task_type == "tunertracker":
        addr = cluster_config["tunertracker"][0]
        command = "python3 -m tvm.exec.rpc_tracker --port={0}".format(addr.get('port'))
        print("DO: " + command)
        os.system(command)
    elif task_type == "tunerserver":
        time.sleep(5)
        addr = cluster_config["tunertracker"][0]
        label = labels_config["tunerserver"]
        command = "python3 -m tvm.exec.rpc_server --tracker={0}:{1} --key={2}".format(addr.get('url'), addr.get('port'), label)
        print("DO: " + command)
        os.system(command)
    elif task_type == "tuner":
        time.sleep(5)
        addr = cluster_config["tunertracker"][0]
        label = labels_config["tunerserver"]
        command = "python3 /home/scripts/auto-tuning.py --tracker {0} --tracker_port {1} --server_key {2}".format(addr.get('url'), addr.get('port'), label)
        print("DO: " + command)
        os.system(command)
    else:
        print("Unknow task type! Error")
