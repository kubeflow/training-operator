# Testing v1alpha2

v1alpha2 is not ready yet but if you still want to try it out you can run it locally by following the instructions below. You need to run it locally because it is not included in Kubeflow yet. Please have a look at [developer_guide.md](../developer_guide.md)

## Create a TFJob

Please see the [example](../examples/v1alpha2/dist-mnist/README.md) to create a TFJob.

## Monitor your job

To get the status of your job

```
kubectl get -o yaml tfjobs $JOB
```

Here is sample output for an example job

```yaml
apiVersion: kubeflow.org/v1alpha2
kind: TFJob
metadata:
  clusterName: ""
  creationTimestamp: 2018-05-10T05:51:10Z
  generation: 1
  name: dist-mnist-for-e2e-test
  namespace: default
  resourceVersion: "606"
  selfLink: /apis/kubeflow.org/v1alpha2/namespaces/default/tfjobs/dist-mnist-for-e2e-test
  uid: 243e12f5-5416-11e8-bb3c-484d7e9d305b
spec:
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
            name: dist-mnist-ps
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
          - args:
            - train_steps
            - "50000"
            image: kubeflow/tf-dist-mnist-test:1.0
            name: dist-mnist-worker
            ports:
            - containerPort: 2222
              name: tfjob-port
            resources: {}
status:
  conditions:
  - lastTransitionTime: 2018-05-10T05:51:10Z
    lastUpdateTime: 2018-05-10T05:51:10Z
    message: TFJob dist-mnist-for-e2e-test is created.
    reason: TFJobCreated
    status: "True"
    type: Created
  - lastTransitionTime: 2018-05-10T05:51:11Z
    lastUpdateTime: 2018-05-10T05:51:11Z
    message: TFJob dist-mnist-for-e2e-test is running.
    reason: TFJobRunning
    status: "True"
    type: Running
  startTime: 2018-05-10T05:51:24Z
  tfReplicaStatuses:
    PS:
      active: 2
    Worker:
      active: 4
```
