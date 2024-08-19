/*
Copyright 2024 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2alpha1

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
)

// +kubebuilder:object:root=true

// ClusterTrainingRuntime represents a training runtime which can be referenced as part of
// `trainingRuntimeRef` API in TrainJob. This resource is a cluster-scoped and can be referenced
// by TrainJob that created in *any* namespace.
type ClusterTrainingRuntime struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired ClusterTrainingRuntime.
	Spec TrainingRuntimeSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterTrainingRuntimeList is a collection of cluster training runtimes.
type ClusterTrainingRuntimeList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ClusterTrainingRuntimes.
	Items []ClusterTrainingRuntime `json:"items"`
}

// +kubebuilder:object:root=true

// TrainingRuntime represents a training runtime which can be referenced as part of
// `trainingRuntimeRef` API in TrainJob. This resource is a namespaced-scoped and can be referenced
// by TrainJob that created in the *same* namespace as the TrainingRuntime.
type TrainingRuntime struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired TrainingRuntime.
	Spec TrainingRuntimeSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// TrainingRuntimeList is a collection of training runtimes.
type TrainingRuntimeList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of TrainingRuntimes.
	Items []TrainingRuntime `json:"items"`
}

// TrainingRuntimeSpec represents a specification of the desired training runtime.
type TrainingRuntimeSpec struct {
	// Configuration for the runtime-specific parameters, such as Torch or MPI.
	MLSpec *MLSpec `json:"mlSpec,omitempty"`

	// Number of training nodes.
	// Defaults to 1.
	NumNodes *int32 `json:"numNodes,omitempty"`

	// JobSet configuration which will be used by TrainJob.
	JobSetSpec *jobsetv1alpha2.JobSetSpec `json:",inline"`

	// Configuration for the PodGroup to enable gang-scheduling via supported plugins.
	PodGroupSpec *PodGroupSpec `json:"podGroupSpec,omitempty"`
}

// PodGroupSpec represents a PodGroup configuration to enable gang-scheduling.
type PodGroupSpec struct {
	// Plugin for the gang-scheduling.
	Plugin GangSchedulerPlugin `json:"plugin"`

	// Time threshold to schedule PodGroup for gang-scheduling.
	ScheduleTimeoutSeconds *string `json:"scheduleTimeoutSeconds,omitempty"`
}

// GangSchedulerPlugin represents one of the supported gang-scheduling plugins.
type GangSchedulerPlugin string

const (
	// Volcano plugin for gang-scheduling.
	GangSchedulerPluginVolcano GangSchedulerPlugin = "volcano"

	// Coscheduling plugin from the Kubernetes scheduler-plugins for gang-scheduling.
	GangSchedulerPluginCoscheduling GangSchedulerPlugin = "coscheduling"
)

// MLSpec represents the runtime-specific configuration for various technologies.
// One of the following specs can be set.
type MLSpec struct {
	// Configuration for the PyTorch runtime.
	TorchSpec *TorchSpec `json:"torchSpec,omitempty"`

	// Configuration for the MPI Runtime.
	MPISpec *MPISpec `json:"mpiSpec,omitempty"`
}

// TorchSpec represents a PyTorch runtime configuration.
type TorchSpec struct {
	// Number of processes per node.
	// This value is inserted into the `--nproc-per-node` argument of the `torchrun` CLI.
	// Supported values: `auto`, `cpu`, `gpu`, or int value.
	// TODO (andreyvelich): Add kubebuilder validation.
	// Defaults to `auto`.
	NumProcPerNode *string `json:"numProcPerNode,omitempty"`

	// Whether to run single-node multi-worker training.
	// This value is inserted into the `--standalone` argument of the `torchrun` CLI.
	// Defaults to false.
	Standalone *bool `json:"standalone,omitempty"`

	// Elastic policy for the PyTorch training.
	ElasticPolicy *TorchElasticPolicy `json:"elasticPolicy,omitempty"`
}

// TorchElasticPolicy represents a configuration for the PyTorch elastic training.
// If this policy is set, the `.spec.numNodes` parameter must be omitted, since min and max node
// is used to configure the `torchrun` CLI argument: `--nnodes=minNodes:maxNodes`.
// Only `c10d` backend is supported for the Rendezvous communication.
type TorchElasticPolicy struct {
	// How many times the training job can be restarted.
	// This value is inserted into the `--max-restarts` argument of the `torchrun` CLI and
	// the `.spec.failurePolicy.maxRestarts` parameter of the training Job.
	MaxRestarts *int32 `json:"maxRestarts,omitempty"`

	// Lower limit for the number of nodes to which training job can scale down.
	MinNodes *int32 `json:"minNodes,omitempty"`

	// Upper limit for the number of nodes to which training job can scale up.
	MaxNodes *int32 `json:"maxNodes,omitempty"`

	// Specification which are used to calculate the desired number of nodes. See the individual
	// metric source types for more information about how each type of metric must respond.
	// The HPA will be created to perform auto-scaling.
	Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`
}

// MPISpec represents a MPI runtime configuration.
type MPISpec struct {
	// Number of processes per node.
	// This value is equal to the number of slots for each node in the hostfile.
	NumProcPerNode *int32 `json:"numProcPerNode,omitempty"`

	// Implementation name for the MPI to create the appropriate hostfile.
	MPIImplementation *MPIImplementation `json:"mpiImplementation"`

	// Directory where SSH keys are mounted.
	SSHAuthMountPath *string `json:"SSHAuthMountPath,omitempty"`

	// Whether to run training process on the launcher Job.
	// Defaults to false.
	RunLauncherAsNode *bool `json:"runLauncherAsNode,omitempty"`
}

// MPIImplementation represents one of the supported MPI implementations.
type MPIImplementation string

const (
	MPIImplementationOpenMPI MPIImplementation = "OpenMPI"
	MPIImplementationIntel   MPIImplementation = "Intel"
	MPIImplementationMPICH   MPIImplementation = "MPICH"
)

func init() {
	SchemeBuilder.Register(&ClusterTrainingRuntime{}, &ClusterTrainingRuntimeList{}, &TrainingRuntime{}, &TrainingRuntimeList{})
}
