# KEP-2170: Kubeflow Training V2 API

## Authors

- Andrey Velichkevich - [@andreyvelich](https://github.com/andreyvelich)
- Yuki Iwai - [@tenzen-y](https://github.com/tenzen-y)

Creation date: 2024-07-16

Google doc: https://bit.ly/3WzjTlw

## Overview

This document discusses the new Kubeflow Training V2 API.

When we built the
[Kubeflow Training Operator a couple of years ago](https://docs.google.com/document/d/1x1JPDQfDMIbnoQRftDH1IzGU0qvHGSU4W6Jl4rJLPhI/edit?usp=sharing),
Kubernetes lacked better features to support distributed machine learning (ML) training, such as
SuccessPolicy and RestartPolicy (FailurePolicy). Recently, the Kubernetes community launched the
working group Batch, and then the working group actively worked on evolving the batch/v1 `Job` API
and built [a new Kubernetes SIGs project: `JobSet`](https://github.com/kubernetes-sigs/jobset) to
manage groups of `Jobs`.

This document consolidates efforts for the Cloud Native ML Training between Kubeflow and Kubernetes
communities.

## Motivation

We often implement features similar to batch/v1 `Job`, such as “suspend”, on the Training Operator
side since the Training Operator creates blocks of plain Pod and Service for each rank once
Kubeflow Jobs are created. However, if we continue taking the same approach to use lowest level
abstractions that introduce redundancy, the maintenance costs will continue to increase.

Replacing repetitive infrastructure layers with `JobSet` would help to avoid redundancy and reduce
developer toil.

Additionally, introducing `JobSet` as an infrastructure layer would allow us to introduce batch
workload features such as
[the PodFailurePolicy](https://kubernetes.io/docs/concepts/workloads/controllers/job/#pod-failure-policy)
and [the PodDisruptionCondition](https://kubernetes.io/docs/concepts/workloads/controllers/job/#handling-pod-and-container-failures)
easily.

Please also see the [Kubernetes JobSet and Kubeflow Training Operator collaboration document](https://docs.google.com/document/d/1C2ev7yRbnMTlQWbQCfX7BCCHcLIAGW2MP9f7YeKl2Ck/edit?usp=sharing).

### User Value

In addition to the above motivation, we will address the following user feedback while implementation:

- Confusion around Workers: https://github.com/kubeflow/training-operator/issues/1790
- Support batch/v1 `Job` features: https://github.com/kubeflow/training-operator/issues/1718
- ExitCodes for PodFailurePolicy: https://github.com/kubeflow/training-operator/issues/1749
- Migrate to MPI V2 API: https://github.com/kubeflow/training-operator/issues/1906

### Personas

We can identify the following personas of Training Operator:

1. **DevOps Engineer**. They are familiar with Kubernetes concepts and they know how to manage the
   Kubernetes workloads. Usually, they are not experts in ML frameworks and ML algorithms.
1. **MLOps Engineer**. They are familiar with ML frameworks and they know how to configure
   distributed PyTorch settings such as rendezvous backends or MPI configuration. Usually, they are
   not experts in Kubernetes and ML algorithms.
1. **Data Scientists**. They create model architectures and advanced ML algorithms to train models.
   They prefer to use Python for their work. They are aware of `torch.nn` APIs, but not with
   `torch.distributed` and Kubernetes concepts to scale model training.

Based on the above personas, we should build an API that everyone will benefit from.

### Goals

- Introduce the `TrainingRuntime` and `ClusterTrainingRuntime` APIs that will store blueprints
  for model training and LLM fine-tuning using various ML frameworks. These runtimes will be built
  on top of `JobSet` APIs with additional functionality for special use-cases.
  For example, training using MPI orchestration.
- Introduce Kubeflow `TrainJob` API that allows to reuse these runtimes and quickly start a new
  training job without understanding complex Kubernetes APIs.
- Update Kubeflow Training SDK to allow data scientists quickly create and monitor `TrainJobs`.
- Create community-supported `ClusterTrainingRuntime` for distributed training with PyTorch and MPI.
- Create community-supported `ClusterTrainingRuntime` for LLM fine-tuning for various foundational
  models (e.g. Mistral, LLama-70b, Gemma-7b).
- Work on the following `JobSet` improvements: https://github.com/kubernetes-sigs/jobset/issues/463
  and https://github.com/kubernetes-sigs/jobset/issues/572

### Non-Goals

- Support MPI V1 implementation.
- Distributed training for TensorFlow, XGboost, JAX, and PaddlePaddle will be added after initial
  implementation.
- Migrate Kubeflow V1 controller to use `JobSet`.

## Design Details

We propose these APIs:

- **`TrainJob`**: A single API which allows data scientists to initiate a training and fine-tuning
  job from the pre-deployed training runtime. It allows users to tweak configurations for their
  training jobs such as model parameters, dataset parameters, or trainer configuration.
  The main goal is to hide unnecessary Kubernetes complexity for data scientists.

- **`TrainingRuntime`** and **`ClusterTrainingRuntime`**: Set of blueprints for how to start various
  types of training or fine-tuning jobs. Runtimes are managed by Platform Engineers and allow them
  to configure infrastructure parameters that are required for the **TrainJob**.
  For example, failure policy or gang-scheduling.

The below diagram shows that platform engineers manage `TrainingRuntime` and data scientists create
`TrainJob`:

![user-roles](./user-roles.drawio.svg)

### Worker and Node Definition

To better understand what does Nodes and Worker mean in the diagram above,
the following table explains naming that each framework or technology uses:

<table>
  <tr>
   <td><strong>ML Framework or Technology</strong>
   </td>
   <td><strong>Definition of a Single Device (GPU)</strong>
   </td>
   <td><strong>Definition of a Single VM</strong>
   </td>
   <td><strong>Start Command</strong>
   </td>
   <td><strong>Reference Docs</strong>
   </td>
  </tr>
  <tr>
   <td>Kubernetes
   </td>
   <td>Container Resource Unit
   </td>
   <td>Pod’s Container
   </td>
   <td>Any
   </td>
   <td><a href="https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes">Resource units</a> in K8s
   </td>
  </tr>
  <tr>
   <td>PyTorch
   </td>
   <td>Worker 
<p>
<code>(--nproc-per-node)</code>
   </td>
   <td>Node
<p>
<code>(--nnodes)</code>
   </td>
   <td><code>torchrun</code>
   </td>
   <td><a href="https://pytorch.org/docs/stable/elastic/run.html">PyTorch Elastic</a>
   </td>
  </tr>
  <tr>
   <td>MPI
   </td>
   <td>Slot
<p>
<code>(--np)</code>
   </td>
   <td>Node 
<p>
<code>(--host)</code>
   </td>
   <td><code>mpirun</code>
   </td>
   <td><a href="https://www.open-mpi.org/doc/v4.0/man1/mpirun.1.php">Reference</a> for OpenMPI
   </td>
  </tr>
  <tr>
   <td>TensorFlow
   </td>
   <td>Worker
   </td>
   <td>Worker Pool 
<p>
<a href="https://cloud.google.com/vertex-ai/docs/training/distributed-training#cluster-spec-format">Cluster Spec</a>
   </td>
   <td><code>python</code>
   </td>
   <td><a href="https://www.tensorflow.org/guide/distributed_training">TensorFlow Distributed</a>
   </td>
  </tr>
  <tr>
   <td>Jax
   </td>
   <td>Process <code>jax.local_devices()</code>
   </td>
   <td>Host
<p>
<code>jax.devices()</code>
   </td>
   <td><code>python</code> or <code>mpirun</code>
   </td>
   <td><a href="https://jax.readthedocs.io/en/latest/multi_process.html">Jax Distributed</a>
   </td>
  </tr>
  <tr>
   <td>PaddlePaddle
   </td>
   <td>Worker
   </td>
   <td>Node
   </td>
   <td><code>python -m paddle.distributed.launch</code>
   </td>
   <td><a href="https://www.paddlepaddle.org.cn/documentation/docs/en/guides/06_distributed_training/cluster_quick_start_en.html">Paddle Distributed</a>
   </td>
  </tr>
  <tr>
   <td>XGBoost
   </td>
   <td>Worker
   </td>
   <td><em>Not Applicable</em>
   </td>
   <td><code>python</code> 
   </td>
   <td><a href="https://github.com/dmlc/xgboost/blob/a5a58102e5e82fa508514c34cd8e5f408dcfd3e1/python-package/xgboost/tracker.py#L17">Rabit Tracker</a> for c10d
   </td>
  </tr>
  <tr>
   <td>DeepSpeed
   </td>
   <td>Slot
   </td>
   <td>Node
<p>
<code>(--num_nodes)</code>
   </td>
   <td><code>deepspeed</code>
   </td>
   <td><a href="https://www.deepspeed.ai/getting-started/#resource-configuration-multi-node">DeepSpeed Distributed</a>
   </td>
  </tr>
</table>

## The TrainJob API

The `TrainJob` exposes APIs that data scientist can override in `TrainingRuntime` to create training job:

```golang
type TrainJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of TrainJob.
	Spec TrainJobSpec `json:"spec"`

	// Status defines the current state of TrainJob.
	Status TrainJobStatus `json:"status,omitempty"`
}

type TrainJobSpec struct {
	// Reference to the Training Runtime.
	TrainingRuntimeRef *TrainingRuntimeRef `json:"trainingRuntimeRef"`

	// Parameters that data scientists can override
	TrainerConfig *TrainerConfig `json:"trainerConfig,omitempty"`

	// Configuration for training dataset
	DatasetConfig *DatasetConfig `json:"datasetConfig,omitempty"`

	// Configuration for the pre-trained model and location for model output
	ModelConfig *ModelConfig `json:"modelConfig,omitempty"`

	// Custom metadata to apply for Job, JobSet, etc.
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type TrainingRuntimeRef struct {
	// Name for the training runtime.
	Name string `json:"name"`
	// Namespace for the runtime.
	// If namespace is set, TrainingRuntime is used. Otherwise, ClusterTrainingRuntime is used.
	Namespace string `json:"namespace,omitempty"`
}

type TrainJobStatus struct {
	// Conditions for the TrainJob
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

```

This table explain rationale for each `TrainJob` parameter:

<table>
  <tr>
   <td><strong>Parameter</strong>
   </td>
   <td><strong>What is it ?</strong>
   </td>
  </tr>
  <tr>
   <td><code>TrainingRuntimeRef</code>
   </td>
   <td>Reference to the existing <code>TrainingRuntime</code> that pre-deployed by platform engineers
   </td>
  </tr>
  <tr>
   <td><code>TrainerConfig</code>
   </td>
   <td>Configuration for the Trainer such as image, number of nodes, accelerators.
   </td>
  </tr>
  <tr>
   <td><code>ModelConfig</code>
   </td>
   <td>Configuration for the pre-trained model and location for model output
   </td>
  </tr>
  <tr>
   <td><code>DatasetConfig</code>
   </td>
   <td>Configuration for the dataset that will be used to train or fine-tune model
   </td>
  </tr>
  <tr>
   <td>Labels and Annotations
   </td>
   <td>Custom metadata that needs to be applied to the <code>TrainJob</code> resources: JobSet, Job, Pods.
   </td>
  </tr>
  <tr>
   <td><code>PodSpecOverrides</code>
   </td>
   <td>Custom overrides that are specific to the <code>TrainJob</code> and need to be applied to the
    <code>TrainJob</code> resources. For example, the user identity. Usually, it is managed by
    custom admission webhooks that inject data to the <code>TrainJob</code> after user creates it
    via Python SDK or <code>kubectl</code>
   </td>
  </tr>
</table>

### Example of TrainJob

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: TrainJob
metadata:
  name: torch-ddp
  namespace: tenant-alpha
spec:
  trainingRuntimeRef:
    name: torch-distributed-multi-node
  trainerConfig:
    image: docker.io/custom-training
    command:
      - torchrun train.py
    numNodes: 5
    resourcesPerNode:
      requests:
        nvidia.com/gpu: 2
```

The above command will be converted as follows:

```bash
torchrun --nnodes=5 --nproc-per-node=2 train.py
```

Additionally, the Kubeflow Training SDK allows to create the above `TrainJob` using the Python API:

```python
def train_func():
    import torch
    class Net(torch.nn.Module):
        """Create the PyTorch Model"""
        ...
    model = Net()

    # Attach model to the distributor
    torch.distributed.init_process_group(backend="nccl")
    model = torch.nn.parallel.DistributedDataParallel(model)

    # Train model
    model.train()

# Use Kubeflow SDK to create TrainJob.
from kubeflow.training import TrainingClient

TrainingClient().train(
    name="torch-ddp",
    func=train_func,
    num_nodes=5,
    resources_per_node={"gpu": 2},
)
```

### Example of LLM Fine-Tuning

This example shows how to create `TrainJob` to fine-tune LLama 7b:

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: TrainJob
metadata:
  name: tune-llama-with-yelp
  namespace: tenant-alpha
spec:
  trainingRuntimeRef:
    name: torch-tune-llama-7b
  datasetConfig:
    s3:
      bucket: datasets
      path: custom-datasets/yelp-review
```

The below diagram shows which resources will be created for LLM fine-tuning with PyTorch:

![trainjob-diagram](./trainjob-diagram.drawio.svg)

### The Trainer Config API

The `TrainerConfig` represents the APIs that data scientists can use to configure trainer settings:

```golang
type TrainerConfig struct {

  // Docker image for the Trainer.
  Image string `json:"image,omitempty"`

  // Command for the training container.
  // Validate that command contains torchrun or mpirun.
  Command []string `json:"command,omitempty"`

  // Args for the training container.
  Args []string `json:"args,omitempty"`

  // Env for the training container.
  Env []corev1.EnvVar `json:"env,omitempty"`

  // Number of training nodes.
  NumNodes *int32 `json:"numNodes,omitempty"`

  // Resource for each node.
  ResourcesPerNode []corev1.resources `json:"resourcesPerNode,omitempty"`

  // Number of processes in a single node.
  // By default this value == number of GPUs in resources limits.
  NumProcPerNode *int32 `json:"numProcPerNode,omitempty"`
}
```

The following table explains how `TrainingRuntime` parameters will be overridden with `TrainerConfig`.

<table>
  <tr>
   <td><strong><code>TrainerConfig</code> Parameter</strong>
   </td>
   <td><strong> <code>TrainingRuntime</code> Parameter</strong>
   </td>
  </tr>
  <tr>
   <td><code>.image</code>
   </td>
   <td><code>.spec.replicatedJobs[name=’Node’].template.spec.template.spec.containers[name=’trainer’].image</code>
   </td>
  </tr>
  <tr>
   <td><code>.command</code>
   </td>
   <td><code>.spec.replicatedJobs[name=’Node’].template.spec.template.spec.containers[name=’trainer’].command</code>
   </td>
  </tr>
  <tr>
   <td><code>.args</code>
   </td>
   <td><code>.spec.replicatedJobs[name=’Node’].template.spec.template.spec.containers[name=’trainer’].args</code>
   </td>
  </tr>
  <tr>
   <td><code>.env</code>
   </td>
   <td><code>.spec.replicatedJobs[name=’Node’].template.spec.template.spec.containers[name=’trainer’].env</code>
   </td>
  </tr>
  <tr>
   <td><code>.numNodes</code>
   </td>
   <td><code>.spec.numNodes</code>
   </td>
  </tr>
  <tr>
   <td><code>.resourcesPerNode</code>
   </td>
   <td><code>.spec.replicatedJobs[name=’Node’].template.spec.template.spec.containers[name=’trainer’].resources</code>
   </td>
  </tr>
</table>

### The Dataset Config API

The `DatasetConfig` represents the APIs that data scientists can use to configure dataset location.

```golang
type DatasetConfig struct {

    // One of the following can be set.
    HFDatasetProvider *kubeflowv1.HFDatasetProvider `json:"huggingface,omitempty"`

    S3DatasetProvider *kubeflowv1.S3DatasetProvider `json:"s3,omitempty"`

    // (Optional). We can support the Iceberg dataset using PyIceberg.
    IcebergDatasetProvider *kubeflowv1.IcebergDatasetProvider `json:"iceberg,omitempty"`
}

type HFDatasetProvider struct {
  // Path to the HF dataset. For example: yelp-review-full
  RepoId string `json:"repoId"`

  // Whether the dataset needs to be splitted: train/val. E.g. split=train[:50000]
  Split string `json:"split,omitempty"`

  // Secret must contain HF_TOKEN secret.
  // If the secret object contains more than one secret, all secrets are passed.
  AccessTokenSecretRef corev1.SecretReference `json:"accessToken,omitempty"`
}


type S3DatasetProvider struct {
  // S3 endpoint.
  EndpointUrl string `json:"endpointUrl,omitempty"`

  // Name of the S3 bucket.
  BucketName string `json:bucketName`

   // Path to the dataset. All files will be downloaded in that path.
  Path string `json:path`

  // Secret must contain AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY secret.
  // If the secret object contains more than one secret, all secrets are passed.
  // Otherwise, IRSA can be used to access S3.
  AccessTokenSecretRef corev1.SecretReference `json:"accessToken,omitempty"`
}
```

The following tables explains how `TrainingRuntime` parameters will be overridden with the
`DatasetConfig`.

All parameters will be set for this container:

```yaml
.spec.replicatedJobs[name='Initializer'].template.spec.template.spec.containers[name=dataset-initializer]
```

#### The HuggingFace Provider

For the HuggingFace provider environment variable `PROVIDER=hf`.

<table>
  <tr>
   <td><strong><code>DatasetConfig</code> Parameter</strong>
   </td>
   <td><strong><code>TrainingRuntime</code> Parameter</strong>
   </td>
  </tr>
  <tr>
   <td><code>.huggingface.repoId</code>
   </td>
   <td><code>.env[REPO_ID]</code>
   </td>
  </tr>
  <tr>
   <td><code>.huggingface.split</code>
   </td>
   <td><code>.env[SPLIT]</code>
   </td>
  </tr>
</table>

#### The S3 Provider

For the S3 provider environment variable `PROVIDER=s3`.

<table>
  <tr>
   <td><strong><code>DatasetConfig</code> Parameter</strong>
   </td>
   <td><strong><code>TrainingRuntime</code> Parameter</strong>
   </td>
  </tr>
  <tr>
   <td><code>.s3.endpointUrl</code>
   </td>
   <td><code>.env[ENDPOINT_URL]</code>
   </td>
  </tr>
  <tr>
   <td><code>.s3.bucketName</code>
   </td>
   <td><code>.env[BUCKET_NAME]</code>
   </td>
  </tr>
  <tr>
   <td><code>.s3.path</code>
   </td>
   <td><code>.env[PATH]</code>
   </td>
  </tr>
</table>

### The Model Config API

The `ModelConfig` represents the APIs that data scientists can use to configure pre-trained model
input and output location.

```golang
type ModelConfig struct {
    // One of the following can be set.
    HFModelProvider *kubeflowv1.HFModelProvider `json:"huggingface,omitempty"`

    // Potential output location for fine-tuned/trained model.
    OutputArtifact string
}


type HFModelProvider struct {
    // Path to the pre-trained model. google-bert/bert-base-uncased
    RepoID string `json:"repoID"`

    // TODO (andreyvelich): Do we want to support any Transformer ?
    TransformerType string `json:"transformerType,omitempty"`

    // Secret must contain HF_TOKEN secret.
    // If the secret object contains more than one secret, all secrets are passed.
    AccessTokenSecretRef corev1.SecretReference `json:"accessToken,omitempty"`
}
```

The following table explains how `TrainingRuntime` parameters will be overridden with `ModelConfig`.

All parameters will be set for this container:

```yaml
.spec.replicatedJobs[name='Initializer'].template.spec.template.spec.containers[name=model-initializer]
```

#### The HuggingFace Provider

For the HuggingFace provider environment variable `PROVIDER=hf`.

<table>
  <tr>
   <td><strong><code>DatasetConfig</code> Parameter</strong>
   </td>
   <td><strong><code>TrainingRuntime</code> Parameter</strong>
   </td>
  </tr>
  <tr>
   <td><code>.huggingface.repoId</code>
   </td>
   <td><code>.env[REPO_ID]</code>
   </td>
  </tr>
  <tr>
   <td><code>.huggingface.split</code>
   </td>
   <td><code>.env[SPLIT]</code>
   </td>
  </tr>
</table>

### The Pod Spec Overrides APIs

The `PodSpecOverrides` represents overrides for the `TrainingRuntime` when `TrainJob` is created.
These parameters can include the user's identity or PVC.

Usually, these parameters should not be configured by the user and should be attached during the
orchestration (e.g. using Kubernetes admission webhooks or custom clients).

In the future, we can add more parameters if we find use-cases when it is required.

```golang
type PodSpecOverride struct {
    // Name of the training replica in the training runtime template to override
    TargetReplicatedJobs []string `json:"targetReplicatedJobs"`

    // Override parameters for Containers.
    Containers []Container `json:"container,omitempty"`

    // Override parameters for InitContainers.
    InitContainer []Container `json:"initContainer,omitempty"`

    // Override parameters for volumes.
    Volumes []corev1.Volume `json:"volume,omitempty"`

    // Custom Service Account
    ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// Override for each container.
// Parameters from TrainerConfig, DatasetConfig, and ModelConfig will take precedence.
type Container struct {

    // Name for the container.
    Name string `json:"name"`

    // Command for the container.
    Command []string `json:"command,omitempty" protobuf:"bytes,3,rep,name=command"`

    // Args for the container.
    Args []string `json:"args,omitempty"`

    // Env for the container.
    Env []corev1.EnvVar `json:"env,omitempty"`

    // Env for the container.
    EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

    // Override parameters for volume mounts.
    VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`
}
```

#### Example of TrainJob with Overrides

This example shows how to override user-identity for sidecar container and add volume to the
trainer container.

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: TrainJob
metadata:
  name: pytorch-distributed
  namespace: tenant-alpha
spec:
  trainingRuntimeRef:
    name: pytorch-distributed-gpu
  trainerConfig:
    image: docker.io/custom-training
  podSpecOverrides:
    - targetReplicatedJobs:
        - initializer
          node
      containers:
        - name: user-identity
          value: 123
        - name: trainer
          volumeMounts:
            - name: user-123-volume
              mountPath: /workspace
      volumes:
        - name: user-123-volume
          persistentVolumeClaim:
            claimName: user-123-volume
```

## The Training Runtime API

The `TrainingRuntime` is the pre-created configurations of model training on the cluster,
representing as blueprints. For example, Elastic PyTorch training, MPI DeepSpeed configuration,
BERT LLM Fine-Tuning.

These blueprints can be deployed within the Training Operator control plane and stored in a Kubeflow
public repository that users can apply to their clusters.

Platform or ML engineers can tweak existing blueprints, based on their requirements. For example,
using custom configurations.

The Kubeflow Training Operator can maintain more Training Runtimes when the community is ready to
support them. For example, runtimes for [Jax](https://jax.readthedocs.io/en/latest/index.html) or
[MLX](https://ml-explore.github.io/mlx/build/html/index.html). Initially, we will support: PyTorch,
MPI, TensorFlow, XGBoost, and PaddlePaddle.

The `TrainingRuntime` is immutable, and so to make a change, a new version of the `TrainingRuntime`
must be created and then changing the `TrainJob` to point to the new version.
This provides control as to how changes to runtimes propagate to existing training jobs.
For example, when training is running for a long time (e.g. 1-2 months).

In the future implementation, we will introduce a revision control mechanism similar to
[Kubernetes Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#updating-a-deployment)
to control versions of `TrainingRuntime` and enable rolling updates.

We are going to create two CRDs: `TrainingRuntime` and `ClusterTrainingRuntime`. These runtimes have
exactly the same APIs, but the first one is the namespace-scoped, the second is the cluster-scoped.
If `trainingRuntimeRef` in `TrainJob` has the namespace, controller will use the `TrainingRuntime`,
otherwise it will use the `ClusterTrainingRuntime`.

```golang
type TrainingRuntime struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    // Framework specific parameters.
    MLSpec *MLSpec `json:"mlSpec,omitempty"`

    // Number of nodes to execute training.
    NumNodes int `json:"numNodes,omitempty"`

    // JobSet spec.
    JobSetSpec *batchv1.JobSetSpec `json:",inline"`

    // For gang-scheduling using volcano or scheduler plugins, supported for all frameworks.
    GangScheduler *kubeflowv1.GangScheduler `json:"gangScheduler,omitempty"`
}

// One of the specs can be selected.
type MLSpec struct {

    // Custom Spec for Torch
    TorchSpec *TorchSpec `json:"torchSpec,omitempty"`

    // Custom Spec for MPI
    MPISpec *MPISpec `json:"mpiSpec,omitempty"`
}
```

### The Gang Scheduler API

Gang scheduler plugin is used to create appropriate `PodGroup` for Volcano or scheduler plugins.

```golang
type GangScheduler struct {
    // Plugin for gang scheduling.
    Plugin *GangSchedulerPlugin `json:plugin,omitempty"`

    // Time threshold to schedule PodGroup for gang scheduling.
    ScheduleTimeoutSeconds string `json:scheduleTimeoutSeconds,omitempty"`
}

type GangSchedulerPlugin string

const (
    GangSchedulerPluginVolcano GangSchedulerPlugin = "volcano"
    GangSchedulerPluginSP      GangSchedulerPlugin = "scheduler-plugins"
)
```

### The Torch Spec API

The `TorchSpec` API represents the configuration for the PyTorch distributed training. This configuration
allows platform engineers to explicitly configure `torchrun` setting.

The distributed parameters are taken from the
[PyTorch distributed launch run](https://github.com/pytorch/pytorch/blob/94dc3253a0fefbfb95fbe467ddd628e4c2eb08d7/torch/distributed/run.py).

For Elastic Training we will always pass the following parameters:

```bash
--rdzv-backend=c10d

--rdzv-id will be set automatically.

--rdzv-endpoint will always point to the node-0 Pod.
```

Since the [etcd and etcd-v2 are legacy rendezvous](https://pytorch.org/docs/stable/elastic/run.html#note-on-rendezvous-backend),
we won't support them in `TorchSpec`. We can introduce them in the future if users will require them.

```golang
// TorchSpec represents the configuration for PyTorch.
type TorchSpec struct {

    // Number of Procs per Node.
    NumProcPerNode int `json:"numProcPerNode,omitempty"`

    // Used for single-node multi-worker training
    Standalone bool `json:"standalone,omitempty"`

    // Torch Elastic Policy.
    ElasticPolicy *TorchElasticPolicy `json:"elasticPolicy,omitempty"`
}

// If the Elastic Policy is set, the numNodes parameter is ignored.
// --nnodes=minNodes:maxNodes
type TorchElasticPolicy struct {

    // The limits to restart TrainJob.
    // Insert it to the JobSet.spec.failurePolicy.maxRestarts
    MaxRestarts *in32 `json:"maxRestarts,omitempty"`

    // Min number of nodes for HPA and torchrun.
    MinNodes *in32 `json:"minNodes,omitempty"`

    // Max number of nodes for HPA and torchrun.
    MaxNodes *in32 `json:"maxNodes,omitempty"`

    // Metrics for scale up and down replicas.
    Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`
}
```

### The MPI Spec API

The `MPISpec` API represents the configuration for training using MPI orchestration.
E.g. creation of host-files and SSH keys. Using MPI might be more efficient for training on HPC
clusters or for some ML frameworks (e.g. [MLX distributed with MPI](https://ml-explore.github.io/mlx/build/html/usage/distributed.html)).

We will fully migrate to the MPI Operator V2 functionality as part of this KEP.
Check [the proposal for the MPI V2 APIs.](https://github.com/kubeflow/mpi-operator/blob/master/proposals/scalable-robust-operator.md)

```golang
type MPISpec struct {
    // Number of Procs per Node.
    NumProcPerNode int `json:"numProcPerNode,omitempty"`

    // MPI Implementation to create appropriate host-files.
    // Can be one of OpenMPI, Intel, or MPICH.
    MPIImplementation MPIImplementation `json:"mpiImplementation,omitempty"`

    // Directory where SSH keys are mounted.
    SSHAuthMountPath string `json:"SSHAuthMountPath,omitempty"`
}

type MPIImplementation string

const (
    MPIImplementationOpenMPI MPIImplementation = "OpenMPI"
    MPIImplementationIntel   MPIImplementation = "Intel"
    MPIImplementationMPICH   MPIImplementation = "MPICH"
)
```

### Supported Runtimes by Community

Kubeflow community are planning to support the following runtimes.

#### PyTorch Distributed Runtime

Initially, we will maintain only multi-node multi-worker runtime and PyTorch Elastic.

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: ClusterTrainingRuntime
metadata:
  name: torch-distributed-multi-node
spec:
  mlSpec:
    torch:
      numProcPerNode: 5
  replicatedJobs:
    - name: node
      template:
        spec:
          template:
            spec:
              containers:
                - name: trainer
                  image: docker.io/kubeflow/pytorch-mnist
                  env:
                    - name: MASTER_ADDR
                      value: "pytorch-node-0-0.pytorch"
                    - name: MASTER_PORT
                      value: 29400
                  command:
                    - torchrun train.py
```

Example of usage:

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: TrainJob
metadata:
  name: torch-test
  namespace: tenant-alpha
spec:
  trainingRuntimeRef:
    name: torch-distributed-multi-node
  trainerConfig:
    resourcesPerNode:
      requests:
        nvidia.com/gpu: 1
    args:
      - num-epochs=5
```

#### PyTorch Elastic Runtime

Training runtime for PyTorch Elastic:

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: ClusterTrainingRuntime
metadata:
  name: torch-distributed-elastic
spec:
  mlSpec:
    torchSpec:
      elasticPolicy:
        minNodes: 5
        maxNodes: 10
        metrics:
          - type: Resource
            resource:
              name: cpu
              target:
                type: Utilization
                averageUtilization: 80
  replicatedJobs:
    - name: node
      template:
        spec:
          template:
            spec:
              containers:
                - name: trainer
                  image: docker.io/kubeflow/pytorch-mnist
                  env:
                    - name: MASTER_ADDR
                      value: "pytorch-node-0-0.pytorch"
                    - name: MASTER_PORT
                      value: 29400
                  command:
                    - torchrun train.py
```

#### Additional PyTorch Runtimes

The following runtimes can be maintained in the future.

Single worker training:

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: ClusterTrainingRuntime
metadata:
  name: torch-simple
spec:
  replicatedJobs:
    - name: node
      template:
        spec:
          template:
            spec:
              containers:
                - name: trainer
                  image: docker.io/kubeflow/pytorch-mnist
                  command:
                    - torchrun train.py
```

Single node multi worker training:

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: ClusterTrainingRuntime
metadata:
  name: torch-distributed-single-worker
spec:
  mlSpec:
    torch:
      numProcPerNode: 5
      standalone: True
  replicatedJobs:
    - name: Node
      template:
        spec:
          template:
            spec:
              containers:
                - name: trainer
                  image: docker.io/kubeflow/pytorch-mnist
                  env:
                    - name: MASTER_ADDR
                      value: "pytorch-node-0-0.pytorch"
                    - name: MASTER_PORT
                      value: 29400
                  command:
                    - torchrun train.py
```

#### LLM Fine-Tuning Runtimes

In the future, we can consider to use [the `torchtune` CLI](https://github.com/pytorch/torchtune/tree/main)
for Fine-Tuning with PyTorch.

##### Llama 7b

The following runtime can be used for Llama 7b model.

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: ClusterTrainingRuntime
metadata:
  name: torch-tune-llama-7b
spec:
  numNodes: 1
  startupPolicy:
    startupPolicyOrder: InOrder
  replicatedJobs:
    - name: Initializer
      template:
        spec:
          template:
            spec:
              containers:
                - name: dataset-initializer
                  image: docker.io/kubeflow/dataset-initializer
                  env:
                    - name: DATASET_PROVIDER
                      value: hf
                    - name: REPO_ID
                      value: tatsu-lab/alpaca
                  volumeMounts:
                    - mountPath: /workspace/dataset
                      name: dataset-initializer
                - name: model-initializer
                  image: docker.io/kubeflow/model-initializer
                  env:
                    - name: MODEL_PROVIDER
                      value: hf
                    - name: REPO_ID
                      value: meta-llama/Llama-2-7b
                    - name: TRANSFORMER_TYPE
                      value: AutoModelForCausalLM
                  volumeMounts:
                    - mountPath: /workspace/model
                      name: model-initializer
              volumes:
                - name: dataset-initializer
                  persistentVolumeClaim:
                    claimName: dataset-initializer
                - name: model-initializer
                  persistentVolumeClaim:
                    claimName: model-initializer
    - name: Node
      template:
        spec:
          template:
            spec:
              containers:
                - name: trainer
                  image: docker.io/kubeflow/llm-trainer
                  env:
                    - name: MASTER_ADDR
                      value: "pytorch-node-0-0.pytorch"
                    - name: MASTER_PORT
                      value: 29400
                    - name: TRANSFORMER_TYPE
                      value: AutoModelForCausalLM
                    - name: LORA_CONFIG
                      value: |
                        {"peft_type": "LORA", "r": 8, "lora_alpha": 16}
                  command:
                    - torchrun hf_llm_training.py
                  resources:
                    limits:
                      nvidia.com/gpu: 2
                  volumeMounts:
                    - mountPath: /workspace/dataset
                      name: dataset-initializer
                    - mountPath: /workspace/model
                      name: model-initializer
              volumes:
                - name: dataset-initializer
                  persistentVolumeClaim:
                    claimName: dataset-initializer
                - name: model-initializer
                  persistentVolumeClaim:
                    claimName: model-initializer
```

##### Gemma 7b

The following runtime can be used for Gemma fine-tuning.

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: ClusterTrainingRuntime
metadata:
  name: torch-tune-gemma-7b
spec:
  numNodes: 1
  startupPolicy:
    startupPolicyOrder: InOrder
  replicatedJobs:
    - name: Initializer
      template:
        spec:
          template:
            spec:
              containers:
                - name: dataset-initializer
                  image: docker.io/kubeflow/dataset-initializer
                  env:
                    - name: DATASET_PROVIDER
                      value: hf
                    - name: REPO_ID
                      value: tatsu-lab/alpaca
                  volumeMounts:
                    - mountPath: /workspace/dataset
                      name: dataset-initializer
                - name: model-initializer
                  image: docker.io/kubeflow/model-initializer
                  env:
                    - name: MODEL_PROVIDER
                      value: hf
                    - name: REPO_ID
                      value: google/gemma-7b
                    - name: TRANSFORMER_TYPE
                      value: AutoModelForCausalLM
                  volumeMounts:
                    - mountPath: /workspace/model
                      name: model-initializer
              volumes:
                - name: dataset-initializer
                  persistentVolumeClaim:
                    claimName: dataset-initializer
                - name: model-initializer
                  persistentVolumeClaim:
                    claimName: model-initializer
    - name: Node
      template:
        spec:
          template:
            spec:
              containers:
                - name: trainer
                  image: docker.io/kubeflow/llm-trainer
                  env:
                    - name: MASTER_ADDR
                      value: "pytorch-node-0-0.pytorch"
                    - name: MASTER_PORT
                      value: 29400
                    - name: TRANSFORMER_TYPE
                      value: AutoModelForCausalLM
                    - name: LORA_CONFIG
                      value: |
                        {"peft_type": "LORA", "r": 8, "lora_alpha": 16}
                  command:
                    - torchrun hf_llm_training.py
                  resources:
                    limits:
                      nvidia.com/gpu: 2
                  volumeMounts:
                    - mountPath: /workspace/dataset
                      name: dataset-initializer
                    - mountPath: /workspace/model
                      name: model-initializer
              volumes:
                - name: dataset-initializer
                  persistentVolumeClaim:
                    claimName: dataset-initializer
                - name: model-initializer
                  persistentVolumeClaim:
                    claimName: model-initializer
```

#### MPI Runtime

For MPI, we can add support the `DeepSpeed` runtimes.

Example of simple OpenMPI runtime:

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: ClusterTrainingRuntime
metadata:
  name: mpi-simple
spec:
  mlSpec:
    mpi:
      mpiImplementation: OpenMPI
      numProcPerNode: 5
  numNodes: 5
  replicatedJobs:
    - name: Launcher
      template:
        spec:
          template:
            spec:
              containers:
                - name: mpi-launcher
                  image: docker.io/mpi-launch
    - name: Node
      template:
        spec:
          template:
            spec:
              containers:
                - name: trainer
                  image: docker.io/mpi-training
                  command:
                    - mpirun -np 2 train.py
```

#### TensorFlow Runtime

_Will be added after initial implementation for PyTorch._

#### XGBoost Runtime

_Will be added after initial implementation for PyTorch._

#### Paddle Runtime

_Will be added after initial implementation for PyTorch._

#### Jax Runtime

_Will be added after initial implementation for PyTorch._

## Migration from Kubeflow Training V1

These API changes will not be compatible with Training Operator V1 APIs. Thus, existing users have
to migrate to the newer APIs. Kubeflow community will provide instructions on how to migrate existing
training jobs to the new APIs.

### PyTorchJob Migration

The following example shows how to migrate from `PyTorchJob` to `TrainingRuntime`:

```yaml
apiVersion: kubeflow.org/v1
kind: PyTorchJob
metadata:
  name: pytorch-simple
  namespace: kubeflow
spec:
  pytorchReplicaSpecs:
    Master:
      replicas: 1
      restartPolicy: OnFailure
      template:
        spec:
          containers:
            - name: pytorch
              image: docker.io/kubeflowkatib/pytorch-mnist:v1beta1-45c5727
              imagePullPolicy: Always
              command:
                - "python3"
                - "/opt/pytorch-mnist/mnist.py"
                - "--epochs=1"
    Worker:
      replicas: 1
      restartPolicy: OnFailure
      template:
        spec:
          containers:
            - name: pytorch
              image: docker.io/kubeflowkatib/pytorch-mnist:v1beta1-45c5727
              imagePullPolicy: Always
              command:
                - "python3"
                - "/opt/pytorch-mnist/mnist.py"
                - "--epochs=1"
```

```yaml
apiVersion: kubeflow.org/v2alpha1
kind: TrainingRuntime
metadata:
  name: torch-distributed-multi-node
spec:
  numNodes: 2
  replicatedJobs:
    - name: node
      template:
        spec:
          template:
            spec:
              containers:
                - name: trainer
                  image: docker.io/kubeflowkatib/pytorch-mnist:v1beta1-45c5727
                  env:
                    - name: MASTER_ADDR
                      value: "pytorch-node-0-0.pytorch"
                    - name: MASTER_PORT
                      value: 29400
                  command:
                    - torchrun train.py
```
