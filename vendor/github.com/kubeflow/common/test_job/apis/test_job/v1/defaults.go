// Copyright 2019 The Kubeflow Authors
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

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
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

// setDefaultPort sets the default ports for container.
func setDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == DefaultContainerName {
			index = i
			break
		}
	}

	hasJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == DefaultPortName {
			hasJobPort = true
			break
		}
	}
	if !hasJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          DefaultPortName,
			ContainerPort: DefaultPort,
		})
	}
}

func setDefaultReplicas(spec *commonv1.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = DefaultRestartPolicy
	}
}

// setTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setTypeNamesToCamelCase(testJob *TestJob) {
	setTypeNameToCamelCase(testJob, TestReplicaTypeWorker)
	setTypeNameToCamelCase(testJob, TestReplicaTypeMaster)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
// E.g. from ps to PS; from WORKER to Worker.
func setTypeNameToCamelCase(testJob *TestJob, typ TestReplicaType) {
	for t := range testJob.Spec.TestReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := testJob.Spec.TestReplicaSpecs[t]
			delete(testJob.Spec.TestReplicaSpecs, t)
			testJob.Spec.TestReplicaSpecs[typ] = spec
			return
		}
	}
}

// SetDefaults_TestJob sets any unspecified values to defaults.
func SetDefaults_TestJob(testjob *TestJob) {
	// Set default cleanpod policy to Running.
	if testjob.Spec.RunPolicy.CleanPodPolicy == nil {
		running := commonv1.CleanPodPolicyRunning
		testjob.Spec.RunPolicy.CleanPodPolicy = &running
	}

	// Update the key of TestReplicaSpecs to camel case.
	setTypeNamesToCamelCase(testjob)

	for _, spec := range testjob.Spec.TestReplicaSpecs {
		// Set default replicas to 1.
		setDefaultReplicas(spec)
		// Set default port to the container.
		setDefaultPort(&spec.Template.Spec)
	}
}
