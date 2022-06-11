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
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// XGBoostJobSpec defines the desired state of XGBoostJob
type XGBoostJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//+kubebuilder:validation:Optional
	RunPolicy commonv1.RunPolicy `json:"runPolicy"`

	XGBReplicaSpecs map[commonv1.ReplicaType]*commonv1.ReplicaSpec `json:"xgbReplicaSpecs"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[-1:].type`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XGBoostJob is the Schema for the xgboostjobs API
// +k8s:openapi-gen=true
type XGBoostJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   XGBoostJobSpec     `json:"spec,omitempty"`
	Status commonv1.JobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XGBoostJobList contains a list of XGBoostJob
type XGBoostJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []XGBoostJob `json:"items"`
}

const (
	// XGBoostReplicaTypeMaster is the type for master replica.
	XGBoostReplicaTypeMaster commonv1.ReplicaType = "Master"

	// XGBoostReplicaTypeWorker is the type for worker replicas.
	XGBoostReplicaTypeWorker commonv1.ReplicaType = "Worker"
)
