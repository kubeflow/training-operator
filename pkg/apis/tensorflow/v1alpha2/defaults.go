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
	"strings"

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

// setDefaultPort sets the default ports for tensorflow container.
func setDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == DefaultContainerName {
			index = i
			break
		}
	}

	hasTFJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == DefaultPortName {
			hasTFJobPort = true
			break
		}
	}
	if !hasTFJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          DefaultPortName,
			ContainerPort: DefaultPort,
		})
	}
}

func setDefaultReplicas(spec *TFReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = DefaultRestartPolicy
	}
}

// setTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setTypeNamesToCamelCase(tfJob *TFJob) {
	setTypeNameToCamelCase(tfJob, TFReplicaTypePS)
	setTypeNameToCamelCase(tfJob, TFReplicaTypeWorker)
	setTypeNameToCamelCase(tfJob, TFReplicaTypeChief)
	setTypeNameToCamelCase(tfJob, TFReplicaTypeEval)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
// E.g. from ps to PS; from WORKER to Worker.
func setTypeNameToCamelCase(tfJob *TFJob, typ TFReplicaType) {
	for t := range tfJob.Spec.TFReplicaSpecs {
		if strings.ToLower(string(t)) == strings.ToLower(string(typ)) && t != typ {
			spec := tfJob.Spec.TFReplicaSpecs[t]
			delete(tfJob.Spec.TFReplicaSpecs, t)
			tfJob.Spec.TFReplicaSpecs[typ] = spec
			return
		}
	}
}

// SetDefaults_TFJob sets any unspecified values to defaults.
func SetDefaults_TFJob(tfjob *TFJob) {
	// Set default cleanpod policy to All.
	if tfjob.Spec.CleanPodPolicy == nil {
		running := CleanPodPolicyRunning
		tfjob.Spec.CleanPodPolicy = &running
	}

	// Update the key of TFReplicaSpecs to camel case.
	setTypeNamesToCamelCase(tfjob)

	for _, spec := range tfjob.Spec.TFReplicaSpecs {
		// Set default replicas to 1.
		setDefaultReplicas(spec)
		// Set default port to tensorFlow container.
		setDefaultPort(&spec.Template.Spec)
	}
}
