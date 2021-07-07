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
	"strings"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	v1 "k8s.io/api/core/v1"
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

// setDefaultPort sets the default ports for pytorch container.
func setDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == DefaultContainerName {
			index = i
			break
		}
	}

	hasPyTorchJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == DefaultPortName {
			hasPyTorchJobPort = true
			break
		}
	}
	if !hasPyTorchJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          DefaultPortName,
			ContainerPort: DefaultPort,
		})
	}
}

func setDefaultReplicas(spec *common.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = DefaultRestartPolicy
	}
}

// setTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setTypeNamesToCamelCase(job *PyTorchJob) {
	setTypeNameToCamelCase(job, PyTorchReplicaTypeMaster)
	setTypeNameToCamelCase(job, PyTorchReplicaTypeWorker)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
func setTypeNameToCamelCase(job *PyTorchJob, typ PyTorchReplicaType) {
	for t := range job.Spec.PyTorchReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := job.Spec.PyTorchReplicaSpecs[t]
			delete(job.Spec.PyTorchReplicaSpecs, t)
			job.Spec.PyTorchReplicaSpecs[typ] = spec
			return
		}
	}
}

// SetDefaults_PyTorchJob sets any unspecified values to defaults.
func SetDefaults_PyTorchJob(job *PyTorchJob) {
	// Set default cleanpod policy to None.
	if job.Spec.CleanPodPolicy == nil {
		policy := common.CleanPodPolicyNone
		job.Spec.CleanPodPolicy = &policy
	}

	// Update the key of PyTorchReplicaSpecs to camel case.
	setTypeNamesToCamelCase(job)

	for rType, spec := range job.Spec.PyTorchReplicaSpecs {
		// Set default replicas to 1.
		setDefaultReplicas(spec)
		if rType == PyTorchReplicaTypeMaster {
			// Set default port to pytorch container of Master.
			setDefaultPort(&spec.Template.Spec)
		}
	}
}
