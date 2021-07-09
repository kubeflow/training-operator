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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName is the group name use in this package.
	GroupName = "kubeflow.org"
	// Kind is the kind name.
	Kind = "TFJob"
	// Version is the version.
	Version = "v1"
	// Plural is the Plural for TFJob.
	Plural = "tfjobs"
	// Singular is the singular for TFJob.
	Singular = "tfjob"
	// TFCRD is the CRD name for TFJob.
	TFCRD = "tfjobs.kubeflow.org"
)

var (
	// SchemeGroupVersion is the group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}
	// SchemeGroupVersionKind is the GroupVersionKind of the resource.
	SchemeGroupVersionKind = SchemeGroupVersion.WithKind(Kind)
)

// Resource takes an unqualified resource and returns a Group-qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
