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
// +resource:path=tfjob

// TFJob represents the configuration of signal TFJob
type TFJob struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the TFJob.
	Spec TFJobSpec `json:"spec,omitempty"`

	// Most recently observed status of the TFJob.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	Status TFJobStatus `json:"status,omitempty"`
}

// TFJobSpec is a desired state description of the TFJob.
type TFJobSpec struct {
	// CleanPodPolicy defines the policy to kill pods after TFJob is
	// succeeded.
	// Default to All.
	CleanPodPolicy *CleanPodPolicy `json:"cleanPodPolicy,omitempty"`

	// TFReplicaSpecs is map of TFReplicaType and TFReplicaSpec
	// specifies the TF replicas to run.
	// For example,
	//   {
	//     "PS": TFReplicaSpec,
	//     "Worker": TFReplicaSpec,
	//   }
	TFReplicaSpecs map[TFReplicaType]*TFReplicaSpec `json:"tfReplicaSpecs"`
}

// TFReplicaSpec is a description of the TFReplica
type TFReplicaSpec struct {
	// Replicas is the desired number of replicas of the given template.
	// If unspecified, defaults to 1.
	Replicas *int32 `json:"replicas,omitempty"`

	// Template is the object that describes the pod that
	// will be created for this TFReplica. RestartPolicy in PodTemplateSpec
	// will be overide by RestartPolicy in TFReplicaSpec
	Template v1.PodTemplateSpec `json:"template,omitempty"`

	// Restart policy for all TFReplicas within the TFJob.
	// One of Always, OnFailure, Never and ExitCode.
	// Default to Never.
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty"`
}

// CleanPodPolicy describes how to deal with pods when the TFJob is finished.
type CleanPodPolicy string

const (
	CleanPodPolicyUndefined CleanPodPolicy = ""
	CleanPodPolicyAll       CleanPodPolicy = "All"
	CleanPodPolicyRunning   CleanPodPolicy = "Running"
	CleanPodPolicyNone      CleanPodPolicy = "None"
)

// RestartPolicy describes how the TFReplicas should be restarted.
// Only one of the following restart policies may be specified.
// If none of the following policies is specified, the default one
// is RestartPolicyAlways.
type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "Always"
	RestartPolicyOnFailure RestartPolicy = "OnFailure"
	RestartPolicyNever     RestartPolicy = "Never"

	// `ExitCode` policy means that user should add exit code by themselves,
	// `tf-operator` will check these exit codes to
	// determine the behavior when an error occurs:
	// - 1-127: permanent error, do not restart.
	// - 128-255: retryable error, will restart the pod.
	RestartPolicyExitCode RestartPolicy = "ExitCode"
)

// TFReplicaType is the type for TFReplica.
type TFReplicaType string

const (
	// TFReplicaTypePS is the type for parameter servers of distributed TensorFlow.
	TFReplicaTypePS TFReplicaType = "PS"

	// TFReplicaTypeWorker is the type for workers of distributed TensorFlow.
	// This is also used for non-distributed TensorFlow.
	TFReplicaTypeWorker TFReplicaType = "Worker"

	// TFReplicaTypeChief is the type for chief worker of distributed TensorFlow.
	// If there is "chief" replica type, it's the "chief worker".
	// Else, worker:0 is the chief worker.
	TFReplicaTypeChief TFReplicaType = "Chief"

	// TFReplicaTypeEval is the type for evaluation replica in TensorFlow.
	TFReplicaTypeEval TFReplicaType = "Evaluator"
)

// TFJobStatus represents the current observed state of the TFJob.
type TFJobStatus struct {
	// Conditions is an array of current observed TFJob conditions.
	Conditions []TFJobCondition `json:"conditions"`

	// TFReplicaStatuses is map of TFReplicaType and TFReplicaStatus,
	// specifies the status of each TFReplica.
	TFReplicaStatuses map[TFReplicaType]*TFReplicaStatus `json:"tfReplicaStatuses"`

	// Represents time when the TFJob was acknowledged by the TFJob controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents time when the TFJob was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Represents last time when the TFJob was reconciled. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// TFReplicaStatus represents the current observed state of the TFReplica.
type TFReplicaStatus struct {
	// The number of actively running pods.
	Active int32 `json:"active,omitempty"`

	// The number of pods which reached phase Succeeded.
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which reached phase Failed.
	Failed int32 `json:"failed,omitempty"`
}

// TFJobCondition describes the state of the TFJob at a certain point.
type TFJobCondition struct {
	// Type of TFJob condition.
	Type TFJobConditionType `json:"type"`
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

// TFJobConditionType defines all kinds of types of TFJobStatus.
type TFJobConditionType string

const (
	// TFJobCreated means the tfjob has been accepted by the system,
	// but one or more of the pods/services has not been started.
	// This includes time before pods being scheduled and launched.
	TFJobCreated TFJobConditionType = "Created"

	// TFJobRunning means all sub-resources (e.g. services/pods) of this TFJob
	// have been successfully scheduled and launched.
	// The training is running without error.
	TFJobRunning TFJobConditionType = "Running"

	// TFJobRestarting means one or more sub-resources (e.g. services/pods) of this TFJob
	// reached phase failed but maybe restarted according to it's restart policy
	// which specified by user in v1.PodTemplateSpec.
	// The training is freezing/pending.
	TFJobRestarting TFJobConditionType = "Restarting"

	// TFJobSucceeded means all sub-resources (e.g. services/pods) of this TFJob
	// reached phase have terminated in success.
	// The training is complete without error.
	TFJobSucceeded TFJobConditionType = "Succeeded"

	// TFJobFailed means one or more sub-resources (e.g. services/pods) of this TFJob
	// reached phase failed with no restarting.
	// The training has failed its execution.
	TFJobFailed TFJobConditionType = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=tfjobs

// TFJobList is a list of TFJobs.
type TFJobList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of TFJobs.
	Items []TFJob `json:"items"`
}
