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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName is the group name use in this package.
	GroupName = "kubeflow.org"
	// Kind is the kind name.
	Kind = "PyTorchJob"
	// GroupVersion is the version.
	Version = "v1"
	// Plural is the Plural for pytorchJob.
	Plural = "pytorchjobs"
	// Singular is the singular for pytorchJob.
	Singular = "pytorchjob"
	// PytorchCRD is the CRD name for PytorchJob.
	PytorchCRD = "pytorchjobs.kubeflow.org"
)

// Resource takes an unqualified resource and returns a Group-qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return GroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&PyTorchJob{},
		&PyTorchJobList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
