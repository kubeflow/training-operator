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
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
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

// setDefaultPort sets the default ports for mxnet container.
func setDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == DefaultContainerName {
			index = i
			break
		}
	}

	hasMXJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == DefaultPortName {
			hasMXJobPort = true
			break
		}
	}
	if !hasMXJobPort {
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
func setTypeNamesToCamelCase(mxJob *MXJob) {
	setTypeNameToCamelCase(mxJob, MXReplicaTypeScheduler)
	setTypeNameToCamelCase(mxJob, MXReplicaTypeServer)
	setTypeNameToCamelCase(mxJob, MXReplicaTypeWorker)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
// E.g. from server to Server; from WORKER to Worker.
func setTypeNameToCamelCase(mxJob *MXJob, typ commonv1.ReplicaType) {
	for t := range mxJob.Spec.MXReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := mxJob.Spec.MXReplicaSpecs[t]
			delete(mxJob.Spec.MXReplicaSpecs, t)
			mxJob.Spec.MXReplicaSpecs[typ] = spec
			return
		}
	}
}

// SetDefaults_MXJob sets any unspecified values to defaults.
func SetDefaults_MXJob(mxjob *MXJob) {
	// Set default cleanpod policy to All.
	if mxjob.Spec.RunPolicy.CleanPodPolicy == nil {
		all := commonv1.CleanPodPolicyAll
		mxjob.Spec.RunPolicy.CleanPodPolicy = &all
	}

	// Update the key of MXReplicaSpecs to camel case.
	setTypeNamesToCamelCase(mxjob)

	for _, spec := range mxjob.Spec.MXReplicaSpecs {
		// Set default replicas to 1.
		setDefaultReplicas(spec)
		// Set default port to mxnet container.
		setDefaultPort(&spec.Template.Spec)
	}
}
