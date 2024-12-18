## An MNIST example with single-program multiple-data (SPMD) data parallelism.

The aim here is to illustrate how to use JAX's [`pmap`](https://jax.readthedocs.io/en/latest/_autosummary/jax.pmap.html) to express and execute
[SPMD](https://jax.readthedocs.io/en/latest/glossary.html#term-SPMD) programs for data parallelism along a batch dimension, while also
minimizing dependencies by avoiding the use of higher-level layers and
optimizers libraries.

Adapted from https://github.com/jax-ml/jax/blob/main/examples/spmd_mnist_classifier_fromscratch.py.

```bash
$ kubectl apply -f examples/jax/jax-dist-spmd-mnist/jaxjob_dist_spmd_mnist_gloo.yaml
```

---

```bash
$ kubectl get pods -n kubeflow -l training.kubeflow.org/job-name=jaxjob-mnist
```

```
NAME                    READY   STATUS      RESTARTS   AGE
jaxjob-mnist-worker-0   0/1     Completed   0          108m
jaxjob-mnist-worker-1   0/1     Completed   0          108m
```

---
```bash
$ PODNAME=$(kubectl get pods -l training.kubeflow.org/job-name=jaxjob-mnist,training.kubeflow.org/replica-type=worker,training.kubeflow.org/replica-index=0 -o
name -n kubeflow)
$ kubectl logs -f ${PODNAME} -n kubeflow
```

```
downloaded https://storage.googleapis.com/cvdf-datasets/mnist/train-images-idx3-ubyte.gz to /tmp/jax_example_data/
downloaded https://storage.googleapis.com/cvdf-datasets/mnist/train-labels-idx1-ubyte.gz to /tmp/jax_example_data/
downloaded https://storage.googleapis.com/cvdf-datasets/mnist/t10k-images-idx3-ubyte.gz to /tmp/jax_example_data/
downloaded https://storage.googleapis.com/cvdf-datasets/mnist/t10k-labels-idx1-ubyte.gz to /tmp/jax_example_data/
JAX global devices:[CpuDevice(id=0), CpuDevice(id=1), CpuDevice(id=2), CpuDevice(id=3), CpuDevice(id=4), CpuDevice(id=5), CpuDevice(id=6), CpuDevice(id=7), CpuDevice(id=131072), CpuDevice(id=131073), CpuDevice(id=131074), CpuDevice(id=131075), CpuDevice(id=131076), CpuDevice(id=131077), CpuDevice(id=131078), CpuDevice(id=131079)]
JAX local devices:[CpuDevice(id=0), CpuDevice(id=1), CpuDevice(id=2), CpuDevice(id=3), CpuDevice(id=4), CpuDevice(id=5), CpuDevice(id=6), CpuDevice(id=7)]
JAX device count:16
JAX local device count:8
JAX process count:2
Epoch 0 in 1809.25 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027
Epoch 1 in 0.51 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027
Epoch 2 in 0.69 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027
Epoch 3 in 0.81 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027
Epoch 4 in 0.91 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027
Epoch 5 in 0.97 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027
Epoch 6 in 1.12 sec
Training set accuracy 0.09035000205039978
Test set accuracy 0.08919999748468399
Epoch 7 in 1.11 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027
Epoch 8 in 1.21 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027
Epoch 9 in 1.29 sec
Training set accuracy 0.09871666878461838
Test set accuracy 0.09799999743700027

```

---

```bash
$ kubectl get -o yaml jaxjobs jaxjob-mnist -n kubeflow
```

```
apiVersion: kubeflow.org/v1
kind: JAXJob
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"kubeflow.org/v1","kind":"JAXJob","metadata":{"annotations":{},"name":"jaxjob-mnist","namespace":"kubeflow"},"spec":{"jaxReplicaSpecs":{"Worker":{"replicas":2,"restartPolicy":"OnFailure","template":{"spec":{"containers":[{"image":"docker.io/sandipanify/jaxjob-spmd-mnist:latest","imagePullPolicy":"Always","name":"jax"}]}}}}}}
  creationTimestamp: "2024-12-18T16:47:28Z"
  generation: 1
  name: jaxjob-mnist
  namespace: kubeflow
  resourceVersion: "3620"
  uid: 15f1db77-3326-405d-95e6-3d9a0d581611
spec:
  jaxReplicaSpecs:
    Worker:
      replicas: 2
      restartPolicy: OnFailure
      template:
        spec:
          containers:
          - image: docker.io/sandipanify/jaxjob-spmd-mnist:latest
            imagePullPolicy: Always
            name: jax
status:
  completionTime: "2024-12-18T17:22:11Z"
  conditions:
  - lastTransitionTime: "2024-12-18T16:47:28Z"
    lastUpdateTime: "2024-12-18T16:47:28Z"
    message: JAXJob jaxjob-mnist is created.
    reason: JAXJobCreated
    status: "True"
    type: Created
  - lastTransitionTime: "2024-12-18T16:50:57Z"
    lastUpdateTime: "2024-12-18T16:50:57Z"
    message: JAXJob kubeflow/jaxjob-mnist is running.
    reason: JAXJobRunning
    status: "False"
    type: Running
  - lastTransitionTime: "2024-12-18T17:22:11Z"
    lastUpdateTime: "2024-12-18T17:22:11Z"
    message: JAXJob kubeflow/jaxjob-mnist successfully completed.
    reason: JAXJobSucceeded
    status: "True"
    type: Succeeded
  replicaStatuses:
    Worker:
      selector: training.kubeflow.org/job-name=jaxjob-mnist,training.kubeflow.org/operator-name=jaxjob-controller,training.kubeflow.org/replica-type=worker
      succeeded: 2
  startTime: "2024-12-18T16:47:28Z"

```
