## Releasing

* The script [release.py](../py/release.py) will build and push a release
  based on the latest green postsubmit.

* The script can be run continuously and will periodically check for new
  postsubmit results.

* [Dockerfile.release](Dockerfile.release) and [releaser.yaml](releaser.yaml)
  provide a Docker image and ReplicaSet spec respectively suitable for running
  the releaser continuously.

* The releaser should be running continuously on a GKE cluster in project
  **tf-on-k8s-releasing**.
