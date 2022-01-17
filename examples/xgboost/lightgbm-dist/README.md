### Distributed Lightgbm Job train

This folder containers Dockerfile and Python scripts to run a distributed Lightgbm training using the XGBoost operator.
The code is based in this [example](https://github.com/microsoft/LightGBM/tree/master/examples/parallel_learning) in the official github repository of the library.


**Build image**
The default image name and tag is `kubeflow/lightgbm-dist-py-test:1.0` respectiveily.

```shell
docker build -f Dockerfile -t kubeflow/lightgbm-dist-py-test:1.0 ./
```

**Start the training**

```
kubectl create -f xgboostjob_v1_lightgbm_dist_training.yaml
```

**Look at the job status**
```
 kubectl get -o yaml XGBoostJob/lightgbm-dist-train-test
 ```
Here is sample output when the job is running. The output result like this

```
apiVersion: xgboostjob.kubeflow.org/v1
kind: XGBoostJob
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"xgboostjob.kubeflow.org/v1","kind":"XGBoostJob","metadata":{"annotations":{},"name":"lightgbm-dist-train-test","namespace":"default"},"spec":{"xgbReplicaSpecs":{"Master":{"replicas":1,"restartPolicy":"Never","template":{"apiVersion":"v1","kind":"Pod","spec":{"containers":[{"args":["--job_type=Train","--boosting_type=gbdt","--objective=binary","--metric=binary_logloss,auc","--metric_freq=1","--is_training_metric=true","--max_bin=255","--data=data/binary.train","--valid_data=data/binary.test","--num_trees=100","--learning_rate=01","--num_leaves=63","--tree_learner=feature","--feature_fraction=0.8","--bagging_freq=5","--bagging_fraction=0.8","--min_data_in_leaf=50","--min_sum_hessian_in_leaf=50","--is_enable_sparse=true","--use_two_round_loading=false","--is_save_binary_file=false"],"image":"kubeflow/lightgbm-dist-py-test:1.0","imagePullPolicy":"Never","name":"xgboostjob","ports":[{"containerPort":9991,"name":"xgboostjob-port"}]}]}}},"Worker":{"replicas":2,"restartPolicy":"ExitCode","template":{"apiVersion":"v1","kind":"Pod","spec":{"containers":[{"args":["--job_type=Train","--boosting_type=gbdt","--objective=binary","--metric=binary_logloss,auc","--metric_freq=1","--is_training_metric=true","--max_bin=255","--data=data/binary.train","--valid_data=data/binary.test","--num_trees=100","--learning_rate=01","--num_leaves=63","--tree_learner=feature","--feature_fraction=0.8","--bagging_freq=5","--bagging_fraction=0.8","--min_data_in_leaf=50","--min_sum_hessian_in_leaf=50","--is_enable_sparse=true","--use_two_round_loading=false","--is_save_binary_file=false"],"image":"kubeflow/lightgbm-dist-py-test:1.0","imagePullPolicy":"Never","name":"xgboostjob","ports":[{"containerPort":9991,"name":"xgboostjob-port"}]}]}}}}}}
  creationTimestamp: "2020-10-14T15:31:23Z"
  generation: 7
  managedFields:
  - apiVersion: xgboostjob.kubeflow.org/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
      f:spec:
        .: {}
        f:xgbReplicaSpecs:
          .: {}
          f:Master:
            .: {}
            f:replicas: {}
            f:restartPolicy: {}
            f:template:
              .: {}
              f:spec: {}
          f:Worker:
            .: {}
            f:replicas: {}
            f:restartPolicy: {}
            f:template:
              .: {}
              f:spec: {}
    manager: kubectl-client-side-apply
    operation: Update
    time: "2020-10-14T15:31:23Z"
  - apiVersion: xgboostjob.kubeflow.org/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:RunPolicy:
          .: {}
          f:cleanPodPolicy: {}
        f:xgbReplicaSpecs:
          f:Master:
            f:template:
              f:metadata:
                .: {}
                f:creationTimestamp: {}
              f:spec:
                f:containers: {}
          f:Worker:
            f:template:
              f:metadata:
                .: {}
                f:creationTimestamp: {}
              f:spec:
                f:containers: {}
      f:status:
        .: {}
        f:completionTime: {}
        f:conditions: {}
        f:replicaStatuses:
          .: {}
          f:Master:
            .: {}
            f:succeeded: {}
          f:Worker:
            .: {}
            f:succeeded: {}
    manager: main
    operation: Update
    time: "2020-10-14T15:34:44Z"
  name: lightgbm-dist-train-test
  namespace: default
  resourceVersion: "38923"
  selfLink: /apis/xgboostjob.kubeflow.org/v1/namespaces/default/xgboostjobs/lightgbm-dist-train-test
  uid: b2b887d0-445b-498b-8852-26c8edc98dc7
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
          - args:
            - --job_type=Train
            - --boosting_type=gbdt
            - --objective=binary
            - --metric=binary_logloss,auc
            - --metric_freq=1
            - --is_training_metric=true
            - --max_bin=255
            - --data=data/binary.train
            - --valid_data=data/binary.test
            - --num_trees=100
            - --learning_rate=01
            - --num_leaves=63
            - --tree_learner=feature
            - --feature_fraction=0.8
            - --bagging_freq=5
            - --bagging_fraction=0.8
            - --min_data_in_leaf=50
            - --min_sum_hessian_in_leaf=50
            - --is_enable_sparse=true
            - --use_two_round_loading=false
            - --is_save_binary_file=false
            image: kubeflow/lightgbm-dist-py-test:1.0
            imagePullPolicy: Never
            name: xgboostjob
            ports:
            - containerPort: 9991
              name: xgboostjob-port
            resources: {}
    Worker:
      replicas: 2
      restartPolicy: ExitCode
      template:
        metadata:
          creationTimestamp: null
        spec:
          containers:
          - args:
            - --job_type=Train
            - --boosting_type=gbdt
            - --objective=binary
            - --metric=binary_logloss,auc
            - --metric_freq=1
            - --is_training_metric=true
            - --max_bin=255
            - --data=data/binary.train
            - --valid_data=data/binary.test
            - --num_trees=100
            - --learning_rate=01
            - --num_leaves=63
            - --tree_learner=feature
            - --feature_fraction=0.8
            - --bagging_freq=5
            - --bagging_fraction=0.8
            - --min_data_in_leaf=50
            - --min_sum_hessian_in_leaf=50
            - --is_enable_sparse=true
            - --use_two_round_loading=false
            - --is_save_binary_file=false
            image: kubeflow/lightgbm-dist-py-test:1.0
            imagePullPolicy: Never
            name: xgboostjob
            ports:
            - containerPort: 9991
              name: xgboostjob-port
            resources: {}
status:
  completionTime: "2020-10-14T15:34:44Z"
  conditions:
  - lastTransitionTime: "2020-10-14T15:31:23Z"
    lastUpdateTime: "2020-10-14T15:31:23Z"
    message: xgboostJob lightgbm-dist-train-test is created.
    reason: XGBoostJobCreated
    status: "True"
    type: Created
  - lastTransitionTime: "2020-10-14T15:31:23Z"
    lastUpdateTime: "2020-10-14T15:31:23Z"
    message: XGBoostJob lightgbm-dist-train-test is running.
    reason: XGBoostJobRunning
    status: "False"
    type: Running
  - lastTransitionTime: "2020-10-14T15:34:44Z"
    lastUpdateTime: "2020-10-14T15:34:44Z"
    message: XGBoostJob lightgbm-dist-train-test is successfully completed.
    reason: XGBoostJobSucceeded
    status: "True"
    type: Succeeded
  replicaStatuses:
    Master:
      succeeded: 1
    Worker:
      succeeded: 2
```