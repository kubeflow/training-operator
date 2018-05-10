# Quick Start (for V1alpha2)

## Installing v1alpha2 CRD and operator

We recommend to deploy Kubeflow in order to use the TFJob v1alpha1 operator. While v1alpha2 is still in experimental, we recommend to run the operator locally. Please have a look at [developer_guide.md](../developer_guide.md)

## Create a TFJob

You create a job by defining a TFJob and then creating it with:

```yaml
apiVersion: "kubeflow.org/v1alpha2"
kind: "TFJob"
metadata:
  name: "dist-mnist-for-e2e-test"
spec:
  tfReplicaSpecs:
    PS:
      replicas: 2
      restartPolicy: Never
      template:
        spec:
          containers:
            - name: dist-mnist-ps
              # See https://github.com/kubeflow/tf-operator/tree/master/test/e2e/dist-mnist to build it.
              image: kubeflow/tf-dist-mnist-test:1.0
    Worker:
      replicas: 4
      restartPolicy: Never
      template:
        spec:
          containers:
            - name: dist-mnist-worker
              # See https://github.com/kubeflow/tf-operator/tree/master/test/e2e/dist-mnist to build it.
              image: kubeflow/tf-dist-mnist-test:1.0
              args: ["train_steps", "50000"]
```

## Monitor your job

To get the status of your job

```
kubectl get tfjobs $JOB
```

Here is sample output for an example job
