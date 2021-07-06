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
	common "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=pytorchjob
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PyTorchJob Represents a PyTorchJob resource.
type PyTorchJob struct {
	// Standard Kubernetes type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard Kubernetes object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired state of the PyTorchJob.
	Spec PyTorchJobSpec `json:"spec,omitempty"`

	// Most recently observed status of the PyTorchJob.
	// Read-only (modified by the system).
	Status common.JobStatus `json:"status,omitempty"`
}

// PyTorchJobSpec is a desired state description of the PyTorchJob.
type PyTorchJobSpec struct {
	// Specifies the duration (in seconds) since startTime during which the job can remain active
	// before it is terminated. Must be a positive integer.
	// This setting applies only to pods where restartPolicy is OnFailure or Always.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	// Number of retries before marking this job as failed.
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// Defines the policy for cleaning up pods after the PyTorchJob completes.
	// Defaults to None.
	CleanPodPolicy *common.CleanPodPolicy `json:"cleanPodPolicy,omitempty"`

	// Defines the TTL for cleaning up finished PyTorchJobs (temporary
	// before Kubernetes adds the cleanup controller).
	// It may take extra ReconcilePeriod seconds for the cleanup, since
	// reconcile gets called periodically.
	// Defaults to infinite.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// A map of PyTorchReplicaType (type) to ReplicaSpec (value). Specifies the PyTorch cluster configuration.
	// For example,
	//   {
	//     "Master": PyTorchReplicaSpec,
	//     "Worker": PyTorchReplicaSpec,
	//   }
	PyTorchReplicaSpecs map[PyTorchReplicaType]*common.ReplicaSpec `json:"pytorchReplicaSpecs"`
}

// PyTorchReplicaType is the type for PyTorchReplica. Can be one of "Master" or "Worker".
type PyTorchReplicaType common.ReplicaType

const (
	// PyTorchReplicaTypeMaster is the type of Master of distributed PyTorch
	PyTorchReplicaTypeMaster PyTorchReplicaType = "Master"

	// PyTorchReplicaTypeWorker is the type for workers of distributed PyTorch.
	PyTorchReplicaTypeWorker PyTorchReplicaType = "Worker"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=pytorchjobs
//+kubebuilder:object:root=true

// PyTorchJobList is a list of PyTorchJobs.
type PyTorchJobList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of PyTorchJobs.
	Items []PyTorchJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PyTorchJob{}, &PyTorchJobList{})
}
