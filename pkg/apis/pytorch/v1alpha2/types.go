// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha2

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=pytorchjob

// PyTorchJob represents the configuration of PyTorchJob
type PyTorchJob struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the PyTorchJob.
	Spec PyTorchJobSpec `json:"spec,omitempty"`

	// Most recently observed status of the PyTorchJob.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	Status PyTorchJobStatus `json:"status,omitempty"`
}

// PyTorchJobSpec is a desired state description of the PyTorchJob.
type PyTorchJobSpec struct {
	// CleanPodPolicy defines the policy to kill pods after PyTorchJob is
	// succeeded.
	// Default to Running.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty"`

	// TTLSecondsAfterFinished is the TTL to clean up pytorch-jobs (temporary
	// before kubernetes adds the cleanup controller).
	// It may take extra ReconcilePeriod seconds for the cleanup, since
	// reconcile gets called periodically.
	// Default to infinite.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinishing,omitempty"`

	// PyTorchReplicaSpecs is map of PyTorchReplicaType and PyTorchReplicaSpec
	// specifies the PyTorch replicas to run.
	// For example,
	//   {
	//     "Master": PyTorchReplicaSpec,
	//     "Worker": PyTorchReplicaSpec,
	//   }
	PyTorchReplicaSpecs map[PyTorchReplicaType]*PyTorchReplicaSpec `json:"pytorchReplicaSpecs"`
}

// PyTorchReplicaSpec is a description of the PyTorchReplica
type PyTorchReplicaSpec struct {
	// Replicas is the desired number of replicas of the given template.
	// If unspecified, defaults to 1.
	Replicas *int32 `json:"replicas,omitempty"`

	// Template is the object that describes the pod that
	// will be created for this PyTorchReplica. RestartPolicy in PodTemplateSpec
	// will be overide by RestartPolicy in PyTorchReplicaSpec
	Template v1.PodTemplateSpec `json:"template,omitempty"`

	// Restart policy for all PyTorchReplicas within the PyTorchJob.
	// One of Always, OnFailure, Never and ExitCode.
	// Default to Never.
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty"`
}

// CleanPodPolicy describes how to deal with pods when the PyTorchJob is finished.
type CleanPodPolicy string

const (
	CleanPodPolicyUndefined CleanPodPolicy = ""
	CleanPodPolicyAll       CleanPodPolicy = "All"
	CleanPodPolicyRunning   CleanPodPolicy = "Running"
	CleanPodPolicyNone      CleanPodPolicy = "None"
)

// RestartPolicy describes how the PyTorchReplicas should be restarted.
// Only one of the following restart policies may be specified.
// If none of the following policies is specified, the default one
// is RestartPolicyAlways.
type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "Always"
	RestartPolicyOnFailure RestartPolicy = "OnFailure"
	RestartPolicyNever     RestartPolicy = "Never"

	// `ExitCode` policy means that user should add exit code by themselves,
	// `pytorch-operator` will check these exit codes to
	// determine the behavior when an error occurs:
	// - 1-127: permanent error, do not restart.
	// - 128-255: retryable error, will restart the pod.
	RestartPolicyExitCode RestartPolicy = "ExitCode"
)

// PyTorchReplicaType is the type for PyTorchReplica.
type PyTorchReplicaType string

const (
	// PyTorchReplicaTypeMaster is the type of Master of distributed PyTorch
	PyTorchReplicaTypeMaster PyTorchReplicaType = "Master"

	// PyTorchReplicaTypeWorker is the type for workers of distributed PyTorch.
	PyTorchReplicaTypeWorker PyTorchReplicaType = "Worker"
)

// PyTorchJobStatus represents the current observed state of the PyTorchJob.
type PyTorchJobStatus struct {
	// Conditions is an array of current observed PyTorchJob conditions.
	Conditions []PyTorchJobCondition `json:"conditions"`

	// PyTorchReplicaStatuses is map of PyTorchReplicaType and PyTorchReplicaStatus,
	// specifies the status of each PyTorchReplica.
	PyTorchReplicaStatuses map[PyTorchReplicaType]*PyTorchReplicaStatus `json:"pytorchReplicaStatuses"`

	// Represents time when the PyTorchJob was acknowledged by the PyTorchJob controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents time when the PyTorchJob was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Represents last time when the PyTorchJob was reconciled. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// PyTorchReplicaStatus represents the current observed state of the PyTorchReplica.
type PyTorchReplicaStatus struct {
	// The number of actively running pods.
	Active int32 `json:"active,omitempty"`

	// The number of pods which reached phase Succeeded.
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which reached phase Failed.
	Failed int32 `json:"failed,omitempty"`
}

// PyTorchJobCondition describes the state of the PyTorchJob at a certain point.
type PyTorchJobCondition struct {
	// Type of PyTorchJob condition.
	Type PyTorchJobConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// PyTorchJobConditionType defines all kinds of types of PyTorchJobStatus.
type PyTorchJobConditionType string

const (
	// PyTorchJobCreated means the pytorchjob has been accepted by the system,
	// but one or more of the pods/services has not been started.
	// This includes time before pods being scheduled and launched.
	PyTorchJobCreated PyTorchJobConditionType = "Created"

	// PyTorchJobRunning means all sub-resources (e.g. services/pods) of this PyTorchJob
	// have been successfully scheduled and launched.
	// The training is running without error.
	PyTorchJobRunning PyTorchJobConditionType = "Running"

	// PyTorchJobRestarting means one or more sub-resources (e.g. services/pods) of this PyTorchJob
	// reached phase failed but maybe restarted according to it's restart policy
	// which specified by user in v1.PodTemplateSpec.
	// The training is freezing/pending.
	PyTorchJobRestarting PyTorchJobConditionType = "Restarting"

	// PyTorchJobSucceeded means all sub-resources (e.g. services/pods) of this PyTorchJob
	// reached phase have terminated in success.
	// The training is complete without error.
	PyTorchJobSucceeded PyTorchJobConditionType = "Succeeded"

	// PyTorchJobFailed means one or more sub-resources (e.g. services/pods) of this PyTorchJob
	// reached phase failed with no restarting.
	// The training has failed its execution.
	PyTorchJobFailed PyTorchJobConditionType = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=pytorchjobs

// PyTorchJobList is a list of PyTorchJobs.
type PyTorchJobList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of PyTorchJobs.
	Items []PyTorchJob `json:"items"`
}
