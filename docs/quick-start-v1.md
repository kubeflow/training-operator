# Testing v1

TFJob is currently in v1. The quick start shows an example of TFJob.
For more details please refer to [developer_guide.md](development/developer_guide.md).

## Create a TFJob

Please see the [example](../examples/tensorflow/dist-mnist/README.md) to create a TFJob.

## Monitor your job

To get the status of your job

```
kubectl get -o yaml tfjobs $JOB
```

Here is sample output for an example job

```yaml
apiVersion: kubeflow.org/v1
kind: TFJob
metadata:
  creationTimestamp: 2019-03-06T09:50:49Z
  generation: 1
  name: dist-mnist-for-e2e-test
  namespace: kubeflow
  resourceVersion: "16575458"
  selfLink: /apis/kubeflow.org/v1/namespaces/kubeflow/tfjobs/dist-mnist-for-e2e-test
  uid: 526545f8-3ff5-11e9-a818-0016ac101ba4
spec:
  cleanPodPolicy: Running
  tfReplicaSpecs:
    PS:
      replicas: 2
      restartPolicy: Never
      template:
        metadata:
          creationTimestamp: null
        spec:
          containers:
            - image: kubeflow/tf-dist-mnist-test:1.0
              name: tensorflow
              ports:
                - containerPort: 2222
                  name: tfjob-port
              resources: {}
    Worker:
      replicas: 4
      restartPolicy: Never
      template:
        metadata:
          creationTimestamp: null
        spec:
          containers:
            - image: kubeflow/tf-dist-mnist-test:1.0
              name: tensorflow
              ports:
                - containerPort: 2222
                  name: tfjob-port
              resources: {}
status:
  conditions:
    - lastTransitionTime: 2019-03-06T09:50:36Z
      lastUpdateTime: 2019-03-06T09:50:36Z
      message: TFJob dist-mnist-for-e2e-test is created.
      reason: TFJobCreated
      status: "True"
      type: Created
    - lastTransitionTime: 2019-03-06T09:50:57Z
      lastUpdateTime: 2019-03-06T09:50:57Z
      message: TFJob dist-mnist-for-e2e-test is running.
      reason: TFJobRunning
      status: "True"
      type: Running
  replicaStatuses:
    PS:
      active: 2
    Worker:
      active: 4
  startTime: 2019-03-06T09:50:48Z
```
