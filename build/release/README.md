## Releasing

* The script [release.py](https://github.com/tensorflow/k8s/blob/master/py/release.py) will build and push a release
  based on the latest green postsubmit.

* The script can be run continuously and will periodically check for new
  postsubmit results.

* [Dockerfile.release](Dockerfile.release) and [releaser.yaml](releaser.yaml)
  provide a Docker image and ReplicaSet spec respectively suitable for running
  the releaser continuously.

* The releaser should be running continuously on a GKE cluster in project
  **tf-on-k8s-releasing**.
    * Logs are available in [stackdriver](https://console.cloud.google.com/logs/viewer?project=tf-on-k8s-releasing&minLogLevel=0&expandAll=false&timestamp=2017-10-30T16:52:47.000000000Z&dateRangeStart=2017-10-30T15:54:39.682Z&interval=PT2H&resource=container%2Fcluster_name%2Freleasing%2Fnamespace_id%2Fdefault)
