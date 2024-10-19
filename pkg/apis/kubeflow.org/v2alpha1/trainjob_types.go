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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
)

const (
	TrainJobKind string = "TrainJob"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TrainJob represents configuration of a training job.
type TrainJob struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired TrainJob.
	Spec TrainJobSpec `json:"spec,omitempty"`

	// Current status of TrainJob.
	Status TrainJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=trainjobs

// +kubebuilder:object:root=true

// TrainJobList is a collection of training jobs.
type TrainJobList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of TrainJobs.
	Items []TrainJob `json:"items"`
}

// TrainJobSpec represents specification of the desired TrainJob.
type TrainJobSpec struct {
	// Reference to the training runtime.
	RuntimeRef RuntimeRef `json:"runtimeRef"`

	// Configuration of the desired trainer.
	Trainer *Trainer `json:"trainer,omitempty"`

	// Configuration of the training dataset.
	DatasetConfig *DatasetConfig `json:"datasetConfig,omitempty"`

	// Configuration of the pre-trained and trained model.
	ModelConfig *ModelConfig `json:"modelConfig,omitempty"`

	// Labels to apply for the derivative JobSet and Jobs.
	// They will be merged with the TrainingRuntime values.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to apply for the derivative JobSet and Jobs.
	// They will be merged with the TrainingRuntime values.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Custom overrides for the training runtime.
	PodSpecOverrides []PodSpecOverride `json:"podSpecOverrides,omitempty"`

	// Whether the controller should suspend the running TrainJob.
	// Defaults to false.
	// +kubebuilder:default=false
	Suspend *bool `json:"suspend,omitempty"`

	// ManagedBy is used to indicate the controller or entity that manages a TrainJob.
	// The value must be either an empty, `kubeflow.org/trainjob-controller` or
	// `kueue.x-k8s.io/multikueue`. The built-in TrainJob controller reconciles TrainJob which
	// don't have this field at all or the field value is the reserved string
	// `kubeflow.org/trainjob-controller`, but delegates reconciling TrainJobs
	// with a 'kueue.x-k8s.io/multikueue' to the Kueue. The field is immutable.
	// Defaults to `kubeflow.org/trainjob-controller`
	// +kubebuilder:default="kubeflow.org/trainjob-controller"
	// +kubebuilder:validation:XValidation:rule="self in ['kubeflow.org/trainjob-controller', 'kueue.x-k8s.io/multikueue']", message="ManagedBy must be kubeflow.org/trainjob-controller or kueue.x-k8s.io/multikueue if set"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="ManagedBy value is immutable"
	ManagedBy *string `json:"managedBy,omitempty"`
}

// RuntimeRef represents the reference to the existing training runtime.
type RuntimeRef struct {
	// Name of the runtime being referenced.
	// When namespaced-scoped TrainingRuntime is used, the TrainJob must have
	// the same namespace as the deployed runtime.
	Name string `json:"name"`

	// APIGroup of the runtime being referenced.
	// Defaults to `kubeflow.org`.
	// +kubebuilder:default="kubeflow.org"
	APIGroup *string `json:"apiGroup,omitempty"`

	// Kind of the runtime being referenced.
	// Defaults to ClusterTrainingRuntime.
	// +kubebuilder:default="ClusterTrainingRuntime"
	Kind *string `json:"kind,omitempty"`
}

// Trainer represents the desired trainer configuration.
// Every training runtime contains `trainer` container which represents Trainer.
type Trainer struct {
	// Docker image for the training container.
	Image *string `json:"image,omitempty"`

	// Entrypoint commands for the training container.
	Command []string `json:"command,omitempty"`

	// Arguments to the entrypoint for the training container.
	Args []string `json:"args,omitempty"`

	// List of environment variables to set in the training container.
	// These values will be merged with the TrainingRuntime's trainer environments.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Number of training nodes.
	// TODO (andreyvelich): Do we want to support dynamic num of nodes in TrainJob for PyTorch elastic: `--nnodes=1:4` ?
	NumNodes *int32 `json:"numNodes,omitempty"`

	// Compute resources for each training node.
	ResourcesPerNode *corev1.ResourceRequirements `json:"resourcesPerNode,omitempty"`

	// Number of processes/workers/slots on every training node.
	// For the Torch runtime: `auto`, `cpu`, `gpu`, or int value can be set.
	// For the MPI runtime only int value can be set.
	NumProcPerNode *string `json:"numProcPerNode,omitempty"`
}

// DatasetConfig represents the desired dataset configuration.
// When this API is used, the training runtime must have
// the `dataset-initializer` container in the `Initializer` Job.
type DatasetConfig struct {
	// Storage uri for the dataset provider.
	StorageUri *string `json:"storageUri,omitempty"`

	// List of environment variables to set in the dataset initializer container.
	// These values will be merged with the TrainingRuntime's dataset initializer environments.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Reference to the TrainJob's secrets to download dataset.
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`
}

// ModelConfig represents the desired model configuration.
type ModelConfig struct {
	// Configuration of the pre-trained model.
	// When this API is used, the training runtime must have
	// the `model-initializer` container in the `Initializer` Job.
	Input *InputModel `json:"input,omitempty"`

	// Configuration of the trained model.
	// When this API is used, the training runtime must have
	// the `model-exporter` container in the `Exporter` Job.
	Output *OutputModel `json:"output,omitempty"`
}

// InputModel represents the desired pre-trained model configuration.
type InputModel struct {
	// Storage uri for the model provider.
	StorageUri *string `json:"storageUri,omitempty"`

	// List of environment variables to set in the model initializer container.
	// These values will be merged with the TrainingRuntime's model initializer environments.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Reference to the TrainJob's secrets to download model.
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`
}

// OutputModel represents the desired trained model configuration.
type OutputModel struct {
	// Storage uri for the model exporter.
	StorageUri *string `json:"storageUri,omitempty"`

	// List of environment variables to set in the model exporter container.
	// These values will be merged with the TrainingRuntime's model exporter environments.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Reference to the TrainJob's secrets to export model.
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`
}

// PodSpecOverride represents the custom overrides that will be applied for the TrainJob's resources.
type PodSpecOverride struct {
	// Names of the training job replicas in the training runtime template to apply the overrides.
	TargetReplicatedJobs []string `json:"targetReplicatedJobs"`

	// Overrides for the containers in the desired job templates.
	Containers []ContainerOverride `json:"containers,omitempty"`

	// Overrides for the init container in the desired job templates.
	InitContainers []ContainerOverride `json:"initContainers,omitempty"`

	// Overrides for the Pod volume configuration.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// Override for the service account.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Override for the node selector to place Pod on the specific mode.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Override for the Pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// ContainerOverrides represents parameters that can be overridden using PodSpecOverrides.
// Parameters from the Trainer, DatasetConfig, and ModelConfig will take precedence.
type ContainerOverride struct {
	// Name for the container. TrainingRuntime must have this container.
	Name string `json:"name"`

	// Entrypoint commands for the training container.
	Command []string `json:"command,omitempty"`

	// Arguments to the entrypoint for the training container.
	Args []string `json:"args,omitempty"`

	// List of environment variables to set in the container.
	// These values will be merged with the TrainingRuntime's environments.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// List of sources to populate environment variables in the container.
	// These   values will be merged with the TrainingRuntime's environments.
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Pod volumes to mount into the container's filesystem.
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

// TrainJobStatus represents the current status of TrainJob.
type TrainJobStatus struct {
	// Conditions for the TrainJob.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ReplicatedJobsStatus tracks the number of Jobs for each replicatedJob in TrainJob.
	ReplicatedJobsStatus []jobsetv1alpha2.ReplicatedJobStatus `json:"replicatedJobsStatus,omitempty"`
}

func init() {
	SchemeBuilder.Register(&TrainJob{}, &TrainJobList{})
}
