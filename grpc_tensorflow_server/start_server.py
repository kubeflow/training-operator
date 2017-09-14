"""
Transform the ClusterSpec from json to piped and calls grpc_tensorflow_server.py
"""
import os
import json

def main():
    tf_config = json.loads(os.environ["TF_CONFIG"])
    cluster_spec = tf_config["cluster"]
    task_id = tf_config["task"]["index"]

    transformed_cluster_spec = ""

    isFirstJob = True
    for job_name in cluster_spec:
        if not isFirstJob:
            #separator between different job kinds
            transformed_cluster_spec += ','
        isFirstJob = False
            
        #separator between job name and it's element
        transformed_cluster_spec += job_name + '|'
        isFirstElement = True
        for el in cluster_spec[job_name]:
            if not isFirstElement:
                #separator between different elements with same job type
                transformed_cluster_spec += ';'
            isFirstElement = False
            transformed_cluster_spec += el

    dir_path = os.path.dirname(os.path.realpath(__file__)) 
    cmd = "python {}/grpc_tensorflow_server.py --cluster_spec '{}' --job_name ps --task_id {}".format(dir_path, transformed_cluster_spec, task_id)
    print(cmd)
    os.system(cmd)

if __name__ == "__main__":
    main()