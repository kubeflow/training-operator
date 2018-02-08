# K8s Custom Resource and Operator For TensorFlow jobs

[![Build Status](https://travis-ci.org/tensorflow/k8s.svg?branch=master)](https://travis-ci.org/tensorflow/k8s)
[![Coverage Status](https://coveralls.io/repos/github/tensorflow/k8s/badge.svg?branch=master)](https://coveralls.io/github/tensorflow/k8s?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/tensorflow/k8s)](https://goreportcard.com/report/github.com/tensorflow/k8s)
[Prow Test Dashboard](https://k8s-testgrid.appspot.com/sig-big-data)
[Prow Jobs](https://prow.k8s.io/?repo=tensorflow%2Fk8s)

## Overview

TFJob provides a Kubernetes custom resource that makes it easy to
run distributed or non-distributed TensorFlow jobs on Kubernetes.

Using a Custom Resource Definition (CRD) gives users the ability to create and manage TF Jobs just like builtin K8s resources. For example to
create a job

```
kubectl create -f examples/tf_job.yaml
```

To list jobs

```
kubectl get tfjobs

NAME          KINDS
example-job   TFJob.v1alpha.kubeflow.org
```

For additional information about motivation and design for the
CRD please refer to
[tf_job_design_doc.md](tf_job_design_doc.md).

### Requirements

TFJob requires Kubernetes >= 1.8
 * CRDs required Kubernetes >= 1.7
 * TFJob depends on Garbage Collection for CRDs which is only supported
   in >= 1.8
 * GPU support is evolving quickly and its best to use Kubernetes 1.8
   to get the latest features.

## Installing the TFJob CRD and operator on your k8s cluster

1. Ensure helm is running on your cluster

   * On GKE with K8s 1.8, follow these
     [instructions](https://docs.helm.sh/using_helm/#tiller-namespaces-and-rbac)
     to setup appropriate service accounts for tiller.

   * Azure K8s clusters should have service accounts configured by
     default for tiller.

1. Deploy the operator

   For RBAC-enabled clusters:
   ```
   CHART=https://storage.googleapis.com/tf-on-k8s-dogfood-releases/latest/tf-job-operator-chart-latest.tgz
   helm install ${CHART} -n tf-job --wait --replace --set rbac.install=true,cloud=<gke or azure>
   ```

   * If you aren't running on GKE or Azure don't set cloud.

   For non-RBAC enabled clusters:
   ```
   CHART=https://storage.googleapis.com/tf-on-k8s-dogfood-releases/latest/tf-job-operator-chart-latest.tgz
   helm install ${CHART} -n tf-job --wait --replace --set cloud=<gke or azure>
   ```

    * The above instructions use the latest release.
    * Releases are versioned
    * You can see a list of versions
    ```
    gsutil ls  gs://tf-on-k8s-dogfood-releases
    ```
    * **Avoiding Breakages**
      * During Alpha there is no guarantees about TFJob API
        compatibility.
      * To avoid being broken by changes you can pin to a particular
        version of the helm chart and control when you upgrade.

1. Make sure the operator is running

    ```
    kubectl get pods

    NAME                               READY     STATUS    RESTARTS   AGE
    tf-job-operator-3083500267-wxj43   1/1       Running   0          48m

    ```

1. Run the helm tests

    ```
    helm test tf-job
    RUNNING: tf-job-tfjob-test-pqxkwk
    PASSED: tf-job-tfjob-test-pqxkwk
    ```

### Installing `tensorflow/k8s`'s Dashboard

> **Caution: the dashboard is in very early development stage!**

`tensorflow/k8s` also includes a dashboard allowing you to monitor and create `TFJobs` through a web UI.
To deploy the dashboard, set `dashboard.install` to `true`.
Note that by default the dashboard will only be accessible from within the cluster or by proxying, as the default `ServiceType` is `ClusterIP`.
If you wish to expose the dashboard through an external IP, set `dashboard.serviceType` to `LoadBalancer`.

So, for example, if you want to enable the dashboard, and also want to expose it externally, you would do:

```
CHART=https://storage.googleapis.com/tf-on-k8s-dogfood-releases/latest/tf-job-operator-chart-latest.tgz
helm install ${CHART} -n tf-job --wait --replace --set cloud=<gke or azure>,dashboard.install=true,dashboard.serviceType=LoadBalancer
```

This sould create a service named `tf-job-dashboard` as well as an additional deployment named `tf-job-dashboard`.


### Configuring the CRD

The CRD must be configured properly to work with your specific Kubernetes cluster.
Since it will be mounting GPU drivers into your pods, the CRD needs to know where to find them on the Kubernetes agents. It also needs to know which environment variable needs to be injected in the pods.

If your Kubernetes cluster is running on GKE or Azure (ACS, AKS, acs-engine) simply pass the provider name to the helm install (or in `tf-job-operator-chart/values.yaml`).

For **GKE**:
```
helm install ${CHART} -n tf-job --wait --replace --set cloud=gke
```

For **Azure**:
```
helm install ${CHART} -n tf-job --wait --replace --set cloud=azure
```

If the cluster is not hosted on GKE or Azure, you will need to specify a custom configuration.
To do so create a `ConfigMap` with your desired settings.

This is the structure of the expected configuration file:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: tf-job-operator-config
  namespace: default
data:
  controller-config-file.yaml: |
    accelerators:
      alpha.kubernetes.io/nvidia-gpu:
        volumes:
          - name: <volume-name> # Desired name of the volume, ex: nvidia-libs
            mountPath: <mount-path> # Path where this should be mounted
            hostPath: <host-path> # Path on the host machine
          - name: <volume2-name> # optional
            mountPath: <mount-path>
            hostPath: <host-path>
        envVars:
          - name: <env-var-name> # Name of the environment variable, ex: LD_LIBRARY_PATH
            value: <env-value> # Value of the environment variable
```

Then simply create the `ConfigMap` and install the Helm chart (**the order matters**) without specifying any cloud provider:

```
kubectl create configmap tf-job-operator-config --from-file <your-configmap-path> --dry-run -o yaml | kubectl replace configmap tf-job-operator-config -f -
helm install ${CHART} -n tf-job --wait --replace
```

Subsequently, any pod requesting a resource of type `alpha.kubernetes.io/nvidia-gpu` will have these Volumes\VolumeMounts and environment variables injected at creation.

## Creating a job

You create a job by defining a TFJob and then creating it with.

```
kubectl create -f https://raw.githubusercontent.com/tensorflow/k8s/master/examples/tf_job.yaml
```

In this case the job spec looks like the following

```
apiVersion: "kubeflow.org/v1alpha1"
kind: "TFJob"
metadata:
  name: "example-job"
spec:
  replicaSpecs:
    - replicas: 1
      tfReplicaType: MASTER
      template:
        spec:
          containers:
            - image: gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff
              name: tensorflow
          restartPolicy: OnFailure
    - replicas: 1
      tfReplicaType: WORKER
      template:
        spec:
          containers:
            - image: gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff
              name: tensorflow
          restartPolicy: OnFailure
    - replicas: 2
      tfReplicaType: PS
```

Each replicaSpec defines a set of TensorFlow processes.
The tfReplicaType defines the semantics for the set of processes.
The semantics are as follows

**master**
  * A job must have 1 and only 1 master
  * The pod must contain a container named tensorflow
  * The overall status of the TFJob is determined by the exit code of the
    tensorflow container
      * 0 = success
      * 1-127 = permanent error
      * 128-255 = retryable error

**worker**
  * A job can have 0 to N workers
  * The pod must contain a container named tensorflow
  * Workers are automatically restarted if they exit

**ps**
  * A job can have 0 to N parameter servers
  * parameter servers are automatically restarted if they exit
  * If you do not specify a container named tensorflow the TFJob
    will automatically add a container to the pod that starts a
    standard TensorFlow gRPC server for each PS.


For each replica you define a **template** which is a K8s
[PodTemplateSpec](https://kubernetes.io/docs/api-reference/v1.8/#podtemplatespec-v1-core).
The template allows you to specify the containers, volumes, etc... that
should be created for each replica.


### Using GPUs

**Note** The use of GPUs and K8s is still in flux.
The following works with GKE & K8s 1.7.2. If this doesn't work on
your setup please consider opening an issue.

Ensure your K8s cluster is properly configured to use GPUs
  * Nodes must have GPUs attached
  * K8s cluster must recognize the nvidia-gpu resource type
  * GPU drivers must be installed on the cluster.
  * Your TFJob controller must be configured to properly attach
    volumes and set environment variables needed for GPUs.

To attach GPUs specify the GPU resource on the container e.g.

```
apiVersion: "kubeflow.org/v1alpha1"
kind: "TFJob"
metadata:
  name: "tf-smoke-gpu"
spec:
  replica_specs:
    - replicas: 1
      tfPort: 2222
      tfReplicaType: MASTER
      template:
        spec:
          containers:
            - image: gcr.io/tf-on-k8s-dogfood/tf_sample_gpu:latest
              name: tensorflow
              resources:
                limits:
                  alpha.kubernetes.io/nvidia-gpu: 1
          restartPolicy: OnFailure
```

Follow TensorFlow's [instructions](https://www.kubeflow.org/tutorials/using_gpu)
for using GPUs.

### Requesting a TensorBoard instance

You can also ask the `TFJob` operator to create a TensorBoard instance
by including a [TensorBoardSpec](https://github.com/tensorflow/k8s/blob/master/pkg/apis/tensorflow/v1alpha1/types.go#L95)
in your job. The table below describes the important fields in
[TensorBoardSpec](https://github.com/tensorflow/k8s/blob/master/pkg/apis/tensorflow/v1alpha1/types.go#L95).

| Name | Description | Required | Default |
|---|---|---|---|
| `logDir` | Specifies the directory where TensorBoard will look to find TensorFlow event files that it can display | Yes | `None` |
| `volumes` | `Volumes` information that will be passed to the TensorBoard `deployment` | No | [] |
| `volumeMounts` | `VolumeMounts` information that will be passed to the TensorBoard `deployment` | No | [] |
| `serviceType` | `ServiceType` information that will be passed to the TensorBoard `service`| No | `ClusterIP` |

#### TensorBoard on Azure

On Azure you can store your event files on an Azure Files and use
volumes to make them available to TensorBoard.

```
apiVersion: "kubeflow.org/v1alpha1"
kind: "TFJob"
metadata:
  name: "tf-smoke-gpu"
spec:
  replica_specs:
    - replicas: 1
      tfReplicaType: MASTER
      template:
        spec:
          containers:
            - image: gcr.io/tf-on-k8s-dogfood/tf_sample_gpu:latest
              name: tensorflow
              resources:
                limits:
                  alpha.kubernetes.io/nvidia-gpu: 1
          restartPolicy: OnFailure
  tensorboard:
    logDir: /tmp/tensorflow
    volumes:
      - name: azurefile
        azureFile:
            secretName: azure-secret
            shareName: data
            readOnly: false
    volumeMounts:
      - mountPath: /tmp/tensorflow
        name: azurefile

```

#### TensorBoard on GKE

On GKE you can store your event files on GCS and TensorBoard/TensorFlow
can read/write directly to GCS.

```
apiVersion: "kubeflow.org/v1alpha1"
kind: "TFJob"
metadata:
  name: "tf-smoke-gpu"
spec:
  replica_specs:
    - replicas: 1
      tfPort: 2222
      tfReplicaType: MASTER
      template:
        spec:
          containers:
            - image: gcr.io/tf-on-k8s-dogfood/tf_sample_gpu:latest
              name: tensorflow
              args:
                - --log_dir=gs://my-bucket/logdir
              resources:
                limits:
                  alpha.kubernetes.io/nvidia-gpu: 1
          restartPolicy: OnFailure
  tensorboard:
    logDir: gs://my-bucket/logdir

```

#### Connecting to TensorBoard

The TFJob operator will create a service named
**tensorboard-$RUNTIME_ID** for your job. You can connect to it
using the Kubernetes API Server proxy as follows

Start the K8s proxy
```
kubectl proxy
```

In a web-browser open up

```
http://${PROXY}:8001/api/v1/proxy/namespaces/default/services/tensorboard-${RUNTIME_ID}:80/
```

Depending on how you configure the service for TensorBoard and cluster
you can make TensorBoard available without using the K8s proxy.

## Monitoring your job

To get the status of your job

```
kubectl get -o yaml tfjobs $JOB
```

Here is sample output for an example job

```
apiVersion: kubeflow.org/v1alpha1
kind: TFJob
metadata:
  clusterName: ""
  creationTimestamp: 2017-10-20T22:27:38Z
  generation: 0
  name: example-job
  namespace: default
  resourceVersion: "1881"
  selfLink: /apis/kubeflow.org/v1alpha1/namespaces/default/tfjobs/example-job
  uid: e11f9577-b5e5-11e7-8522-42010a8e01a4
spec:
  RuntimeId: 76no
  replicaSpecs:
  - replicas: 1
    template:
      metadata:
        creationTimestamp: null
      spec:
        containers:
        - image: gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff
          name: tensorflow
          resources: {}
        restartPolicy: OnFailure
    tfPort: 2222
    tfReplicaType: MASTER
  - replicas: 1
    template:
      metadata:
        creationTimestamp: null
      spec:
        containers:
        - image: gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff
          name: tensorflow
          resources: {}
        restartPolicy: OnFailure
    tfPort: 2222
    tfReplicaType: WORKER
  - replicas: 2
    template:
      metadata:
        creationTimestamp: null
      spec:
        containers:
        - image: tensorflow/tensorflow:1.3.0
          name: tensorflow
          resources: {}
          volumeMounts:
          - mountPath: /ps-server
            name: ps-config-volume
        restartPolicy: OnFailure
    tfPort: 2222
    tfReplicaType: PS
  tensorboard:
    logDir: /tmp/tensorflow
    serviceType: ""
    volumeMounts: null
    volumes: null
  tfImage: tensorflow/tensorflow:1.3.0
status:
  conditions: null
  controlPaused: false
  phase: Done
  reason: ""
  replicaStatuses:
  - ReplicasStates:
      Succeeded: 1
    state: Succeeded
    tf_replica_type: MASTER
  - ReplicasStates:
      Running: 1
    state: Running
    tf_replica_type: WORKER
  - ReplicasStates:
      Running: 2
    state: Running
    tf_replica_type: PS
  state: Succeeded

```

The first thing to note is the **RuntimeId**. This is a random unique
string which is used to give names to all the K8s resouces
(e.g Job controllers & services) that are created by the TFJob.

As with other K8s resources status provides information about the state
of the resource.

**phase** - Indicates the phase of a job and will be one of
 - Creating
 - Running
 - CleanUp
 - Failed
 - Done

**state** - Provides the overall status of the job and will be one of
  - Running
  - Succeeded
  - Failed

For each replica type in the job, there will be a ReplicaStatus that
provides the number of replicas of that type in each state.

For each replica type, the job creates a set of K8s
[Job Controllers](https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/)
named

```
${REPLICA-TYPE}-${RUNTIME_ID}-${INDEX}
```

For example, if you have 2 parameter servers and runtime id 76n0 TFJob
will create the jobs

```
ps-76no-0
ps-76no-1
```

### TensorFlow Logs

Logging follows standard K8s logging practices.

You can use kubectl to get standard output/error for any of
your containers.

First find the pod created by the job controller for the replica of
index. Pods will be named

```
${REPLICA-TYPE}-${RUNTIME_ID}-${INDEX}-${RANDOM}
```

where RANDOM is a unique id generated by K8s to uniquely identify each
pod.

Once you've identified your pod you can get the logs using kubectl.

```
kubectl logs ${REPLICA-TYPE}-${RUNTIME_ID}-${INDEX}-${RADNOM}
```

If your cluster takes advantage of K8s
[logging infrastructure](https://kubernetes.io/docs/concepts/cluster-administration/logging/)
then your logs may also be shipped to an appropriate data store for
further analysis.

#### GKE

The default on GKE is send logs to
[Stackdriver logging](https://cloud.google.com/logging/docs/).

To get the logs for a particular pod you can use the following
advanced filter in Stackdriver logging's search UI.

```
resource.type="container"
resource.labels.pod_id=${POD_NAME}
```

where ${POD_NAME} is the name of the pod.

**Tip** If you don't know the id of the pod, just enter the RuntimeId
for your job into the Stackdriver logging search UI. This will find all
log entries with the RuntimeId anywhere in the log entry. Since the
RuntimeId is a random string, the only matches will be the log entries
for your job.

**Tip** If your program outputs an easily searchable log message with
the replica type and index then you can search for this log message
and use it to determine the ${POD_NAME} for a particular pod; e.g

```
cluster_json = os.getenv('TF_CONFIG')
cluster = json.loads(cluster)
logging.info("REPLICA_TYPE=%s,REPLICA_INDEX=%s", cluster["task"]["type"], cluster["task"]["index"])
```

This would log a message like

```
REPLICA_TYPE=worker,REPLICA_INDEX=0
```

which you could then search for in the StackDriver UI. Once you find
the entry you can expand it to see **resource.labels.pod_id**.


## Contributing

Please refer to the [developer_guide](developer_guide.md)
