### Distributed mnist model for e2e test

This folder containers Dockerfile and distributed mnist model for e2e test.

**Build Image**

The default image name and tag is `kubeflow/tf-dist-mnist-test:1.0`.

```shell
docker build -f Dockerfile -t kubeflow/tf-dist-mnist-test:1.0 ./
```

**Create TFJob YAML**

```yaml
apiVersion: "kubeflow.org/v1alpha2"
kind: "TFJob"
metadata:
  name: "dist-mnist-for-e2e-test"
spec:
  tfReplicaSpecs:
    PS:
      replicas: 2
      restartPolicy: OnFailure
      template:
        spec:
          containers:
            - name: dist-mnist-ps
              image: kubeflow/tf-dist-mnist-test:1.0
    Worker:
      replicas: 4
      restartPolicy: OnFailure
      template:
        spec:
          containers:
            - name: dist-mnist-worker
              image: kubeflow/tf-dist-mnist-test:1.0
              args: ["train_steps", "50000"]
```
