### Distributed send/recv e2e test for xgboost rabit

This folder containers Dockerfile and distributed send/recv test.


**Start and test XGBoost Rabit tracker**

```
kubectl create -f xgboostjob_v1alpha1_rabit_test.yaml
```

**Look at the job status**
```
 kubectl get -o yaml XGBoostJob/xgboost-dist-test
 ```
Here is sample output when the job is running. The output result like this
```
apiVersion: kubeflow.org/v1
kind: XGBoostJob
metadata:
  creationTimestamp: "2019-06-21T03:32:57Z"
  generation: 7
  name: xgboost-dist-test
  namespace: default
  resourceVersion: "258466"
  uid: 431dc182-93d5-11e9-bbab-080027dfbfe2
spec:
  RunPolicy:
    cleanPodPolicy: None
  xgbReplicaSpecs:
    Master:
      replicas: 1
      restartPolicy: Never
      template:
        metadata:
          creationTimestamp: null
        spec:
          containers:
          - image: docker.io/kubeflow/xgboost-dist-rabit-test:latest
            imagePullPolicy: Always
            name: xgboostjob
            ports:
            - containerPort: 9991
              name: xgboostjob-port
            resources: {}
    Worker:
      replicas: 2
      restartPolicy: Never
      template:
        metadata:
          creationTimestamp: null
        spec:
          containers:
          - image: docker.io/kubeflow/xgboost-dist-rabit-test:latest
            imagePullPolicy: Always
            name: xgboostjob
            ports:
            - containerPort: 9991
              name: xgboostjob-port
            resources: {}
status:
  completionTime: "2019-06-21T03:33:03Z"
  conditions:
  - lastTransitionTime: "2019-06-21T03:32:57Z"
    lastUpdateTime: "2019-06-21T03:32:57Z"
    message: xgboostJob xgboost-dist-test is created.
    reason: XGBoostJobCreated
    status: "True"
    type: Created
  - lastTransitionTime: "2019-06-21T03:32:57Z"
    lastUpdateTime: "2019-06-21T03:32:57Z"
    message: XGBoostJob xgboost-dist-test is running.
    reason: XGBoostJobRunning
    status: "False"
    type: Running
  - lastTransitionTime: "2019-06-21T03:33:03Z"
    lastUpdateTime: "2019-06-21T03:33:03Z"
    message: XGBoostJob xgboost-dist-test is successfully completed.
    reason: XGBoostJobSucceeded
    status: "True"
    type: Succeeded
  replicaStatuses:
    Master:
      succeeded: 1
    Worker:
      succeeded: 2
```
 


