// Copyright 2020 The Kubeflow Authors
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

package v1

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=tfjob

// TFJob represents a TFJob resource.
type TFJob struct {
	// Standard Kubernetes type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard Kubernetes object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired state of the TFJob.
	// +optional
	Spec TFJobSpec `json:"spec,omitempty"`

	// Most recently observed status of the TFJob.
	// Populated by the system.
	// Read-only.
	// +optional
	Status commonv1.JobStatus `json:"status,omitempty"`
}

// TFJobSpec is a desired state description of the TFJob.
type TFJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	RunPolicy commonv1.RunPolicy `json:"runPolicy,inline"`

	// SuccessPolicy defines the policy to mark the TFJob as succeeded.
	// Default to "", using the default rules.
	// +optional
	SuccessPolicy *SuccessPolicy `json:"successPolicy,omitempty"`

	// A map of TFReplicaType (type) to ReplicaSpec (value). Specifies the TF cluster configuration.
	// For example,
	//   {
	//     "PS": ReplicaSpec,
	//     "Worker": ReplicaSpec,
	//   }
	TFReplicaSpecs map[commonv1.ReplicaType]*commonv1.ReplicaSpec `json:"tfReplicaSpecs"`

	// // A switch to enable dynamic worker
	EnableDynamicWorker bool `json:"enableDynamicWorker,omitempty"`
}

// TFReplicaType is the type for TFReplica. Can be one of: "Chief"/"Master" (semantically equivalent),
// "Worker", "PS", or "Evaluator".

const (
	// TFReplicaTypePS is the type for parameter servers of distributed TensorFlow.
	TFReplicaTypePS commonv1.ReplicaType = "PS"

	// TFReplicaTypeWorker is the type for workers of distributed TensorFlow.
	// This is also used for non-distributed TensorFlow.
	TFReplicaTypeWorker commonv1.ReplicaType = "Worker"

	// TFReplicaTypeChief is the type for chief worker of distributed TensorFlow.
	// If there is "chief" replica type, it's the "chief worker".
	// Else, worker:0 is the chief worker.
	TFReplicaTypeChief commonv1.ReplicaType = "Chief"

	// TFReplicaTypeMaster is the type for master worker of distributed TensorFlow.
	// This is similar to chief, and kept just for backwards compatibility.
	TFReplicaTypeMaster commonv1.ReplicaType = "Master"

	// TFReplicaTypeEval is the type for evaluation replica in TensorFlow.
	TFReplicaTypeEval commonv1.ReplicaType = "Evaluator"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=tfjobs

// TFJobList is a list of TFJobs.
type TFJobList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of TFJobs.
	Items []TFJob `json:"items"`
}
