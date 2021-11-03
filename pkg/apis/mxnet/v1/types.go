// Copyright 2021 The Kubeflow Authors
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

// MXJobSpec defines the desired state of MXJob
type MXJobSpec struct {
	// RunPolicy encapsulates various runtime policies of the distributed training
	// job, for example how to clean up resources and how long the job can stay
	// active.
	//+kubebuilder:validation:Optional
	RunPolicy common.RunPolicy `json:"runPolicy"`

	// JobMode specify the kind of MXjob to do. Different mode may have
	// different MXReplicaSpecs request
	JobMode JobModeType `json:"jobMode"`

	// MXReplicaSpecs is map of common.ReplicaType and common.ReplicaSpec
	// specifies the MX replicas to run.
	// For example,
	//   {
	//     "Scheduler": common.ReplicaSpec,
	//     "Server": common.ReplicaSpec,
	//     "Worker": common.ReplicaSpec,
	//   }
	MXReplicaSpecs map[common.ReplicaType]*common.ReplicaSpec `json:"mxReplicaSpecs"`
}

// JobModeType id the type for JobMode
type JobModeType string

const (
	// Train Mode, in this mode requested MXReplicaSpecs need
	// has Server, Scheduler, Worker
	MXTrain JobModeType = "MXTrain"

	// Tune Mode, in this mode requested MXReplicaSpecs need
	// has Tuner
	MXTune JobModeType = "MXTune"
)

const (
	// MXReplicaTypeScheduler is the type for scheduler replica in MXNet.
	MXReplicaTypeScheduler common.ReplicaType = "Scheduler"

	// MXReplicaTypeServer is the type for parameter servers of distributed MXNet.
	MXReplicaTypeServer common.ReplicaType = "Server"

	// MXReplicaTypeWorker is the type for workers of distributed MXNet.
	// This is also used for non-distributed MXNet.
	MXReplicaTypeWorker common.ReplicaType = "Worker"

	// MXReplicaTypeTunerTracker
	// This the auto-tuning tracker e.g. autotvm tracker, it will dispatch tuning task to TunerServer
	MXReplicaTypeTunerTracker common.ReplicaType = "TunerTracker"

	// MXReplicaTypeTunerServer
	MXReplicaTypeTunerServer common.ReplicaType = "TunerServer"

	// MXReplicaTuner is the type for auto-tuning of distributed MXNet.
	// This is also used for non-distributed MXNet.
	MXReplicaTypeTuner common.ReplicaType = "Tuner"
)

// MXJobStatus defines the observed state of MXJob
type MXJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=mxjob

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// MXJob is the Schema for the mxjobs API
type MXJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MXJobSpec        `json:"spec,omitempty"`
	Status common.JobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MXJobList contains a list of MXJob
type MXJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MXJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MXJob{}, &MXJobList{})
	SchemeBuilder.SchemeBuilder.Register(addDefaultingFuncs)
}
