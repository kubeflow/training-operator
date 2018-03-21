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
	"k8s.io/apimachinery/pkg/runtime"
)

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func setDefaultPort(spec *v1.PodSpec) {
	for i := range spec.Containers {
		if len(spec.Containers[i].Ports) == 0 {
			spec.Containers[i].Ports = append(spec.Containers[i].Ports, v1.ContainerPort{
				Name:          defaultPortName,
				ContainerPort: defaultPort,
			})
		}
	}
}

func setDefaultReplicas(spec *TFReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
}

// SetDefaults_TFJob sets any unspecified values to defaults.
func SetDefaults_TFJob(tfjob *TFJob) {
	for _, spec := range tfjob.Spec.TFReplicaSpecs {
		setDefaultReplicas(spec)
		setDefaultPort(&spec.Template.Spec)
	}
}
