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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GenericJobSpec defines the desired state of GenericJob
type GenericJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+kubebuilder:validation:Optional
	RunPolicy common.RunPolicy `json:"runPolicy"`

	ReplicaSpecs map[common.ReplicaType]*common.ReplicaSpec `json:"replicaSpecs"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GenericJob is the Schema for the genericjobs API
type GenericJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GenericJobSpec   `json:"spec,omitempty"`
	Status common.JobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GenericJobList contains a list of GenericJob
type GenericJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GenericJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GenericJob{}, &GenericJobList{})
}
