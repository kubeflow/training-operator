// Copyright 2019 The Kubeflow Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"strings"

	v1 "k8s.io/api/core/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

// setDefaultsTypeLauncher sets the default value to launcher.
func setMPIDefaultsTypeLauncher(spec *commonv1.ReplicaSpec) {
	if spec != nil && spec.RestartPolicy == "" {
		spec.RestartPolicy = MPIDefaultRestartPolicy
	}
}

// setDefaultsTypeWorker sets the default value to worker.
func setMPIDefaultsTypeWorker(spec *commonv1.ReplicaSpec) {
	if spec != nil && spec.RestartPolicy == "" {
		spec.RestartPolicy = MPIDefaultRestartPolicy
	}
}

func SetDefaults_MPIJob(mpiJob *MPIJob) {
	// Set default CleanPodPolicy to None when neither fields specified.
	if mpiJob.Spec.CleanPodPolicy == nil && mpiJob.Spec.RunPolicy.CleanPodPolicy == nil {
		none := commonv1.CleanPodPolicyNone
		mpiJob.Spec.CleanPodPolicy = &none
		mpiJob.Spec.RunPolicy.CleanPodPolicy = &none
	}

	// set default to Launcher
	setMPIDefaultsTypeLauncher(mpiJob.Spec.MPIReplicaSpecs[MPIReplicaTypeLauncher])

	// set default to Worker
	setMPIDefaultsTypeWorker(mpiJob.Spec.MPIReplicaSpecs[MPIReplicaTypeWorker])
}

// setDefaultPort sets the default ports for mxnet container.
func setMXDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == MXDefaultContainerName {
			index = i
			break
		}
	}

	hasMXJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == MXDefaultPortName {
			hasMXJobPort = true
			break
		}
	}
	if !hasMXJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          MXDefaultPortName,
			ContainerPort: MXDefaultPort,
		})
	}
}

func setMXDefaultReplicas(spec *commonv1.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = MXDefaultRestartPolicy
	}
}

// setTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setMXTypeNamesToCamelCase(mxJob *MXJob) {
	setMXTypeNameToCamelCase(mxJob, MXReplicaTypeScheduler)
	setMXTypeNameToCamelCase(mxJob, MXReplicaTypeServer)
	setMXTypeNameToCamelCase(mxJob, MXReplicaTypeWorker)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
// E.g. from server to Server; from WORKER to Worker.
func setMXTypeNameToCamelCase(mxJob *MXJob, typ commonv1.ReplicaType) {
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
	setMXTypeNamesToCamelCase(mxjob)

	for _, spec := range mxjob.Spec.MXReplicaSpecs {
		// Set default replicas to 1.
		setMXDefaultReplicas(spec)
		// Set default port to mxnet container.
		setMXDefaultPort(&spec.Template.Spec)
	}
}

// setDefaultPort sets the default ports for pytorch container.
func setPyTorchDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == PyTorchDefaultContainerName {
			index = i
			break
		}
	}

	hasPyTorchJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == PyTorchDefaultPortName {
			hasPyTorchJobPort = true
			break
		}
	}
	if !hasPyTorchJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          PyTorchDefaultPortName,
			ContainerPort: PyTorchDefaultPort,
		})
	}
}

func setPyTorchElasticPolicy(job *PyTorchJob) {
	if job.Spec.ElasticPolicy != nil {
		if job.Spec.ElasticPolicy.MaxReplicas != nil &&
			job.Spec.ElasticPolicy.MinReplicas != nil {
			return
		} else if job.Spec.ElasticPolicy.MaxReplicas != nil {
			// Set MinRepliacs to elasticPolicy.MaxReplicas.
			job.Spec.ElasticPolicy.MinReplicas = job.Spec.ElasticPolicy.MaxReplicas
		} else if job.Spec.ElasticPolicy.MinReplicas != nil {
			job.Spec.ElasticPolicy.MaxReplicas = job.Spec.ElasticPolicy.MinReplicas
		} else {
			workerReplicas := job.Spec.PyTorchReplicaSpecs[PyTorchReplicaTypeWorker].Replicas
			// Set Min and Max to worker.spec.Replicas.
			job.Spec.ElasticPolicy.MaxReplicas = workerReplicas
			job.Spec.ElasticPolicy.MinReplicas = workerReplicas
		}
	}
}

func setPyTorchDefaultReplicas(spec *commonv1.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
}

func setPyTorchDefaultRestartPolicy(spec *commonv1.ReplicaSpec) {
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = PyTorchDefaultRestartPolicy
	}
}

// setTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setPyTorchTypeNamesToCamelCase(job *PyTorchJob) {
	setPyTorchTypeNameToCamelCase(job, PyTorchReplicaTypeMaster)
	setPyTorchTypeNameToCamelCase(job, PyTorchReplicaTypeWorker)
}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
func setPyTorchTypeNameToCamelCase(job *PyTorchJob, typ commonv1.ReplicaType) {
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
	if job.Spec.RunPolicy.CleanPodPolicy == nil {
		policy := commonv1.CleanPodPolicyNone
		job.Spec.RunPolicy.CleanPodPolicy = &policy
	}

	// Update the key of PyTorchReplicaSpecs to camel case.
	setPyTorchTypeNamesToCamelCase(job)

	for _, spec := range job.Spec.PyTorchReplicaSpecs {
		setPyTorchDefaultReplicas(spec)
		setPyTorchDefaultRestartPolicy(spec)
		setPyTorchDefaultPort(&spec.Template.Spec)
	}
	// Set default elastic policy.
	setPyTorchElasticPolicy(job)
}

// setDefaultPort sets the default ports for tensorflow container.
func setTFDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == TFDefaultContainerName {
			index = i
			break
		}
	}

	hasTFJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == TFDefaultPortName {
			hasTFJobPort = true
			break
		}
	}
	if !hasTFJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          TFDefaultPortName,
			ContainerPort: TFDefaultPort,
		})
	}
}

func setTFDefaultReplicas(spec *commonv1.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = TFDefaultRestartPolicy
	}
}

// setTFTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setTFTypeNamesToCamelCase(tfJob *TFJob) {
	setTFTypeNameToCamelCase(tfJob, TFReplicaTypePS)
	setTFTypeNameToCamelCase(tfJob, TFReplicaTypeWorker)
	setTFTypeNameToCamelCase(tfJob, TFReplicaTypeChief)
	setTFTypeNameToCamelCase(tfJob, TFReplicaTypeMaster)
	setTFTypeNameToCamelCase(tfJob, TFReplicaTypeEval)
}

// setTFTypeNameToCamelCase sets the name of the replica type from any case to correct case.
// E.g. from ps to PS; from WORKER to Worker.
func setTFTypeNameToCamelCase(tfJob *TFJob, typ commonv1.ReplicaType) {
	for t := range tfJob.Spec.TFReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := tfJob.Spec.TFReplicaSpecs[t]
			delete(tfJob.Spec.TFReplicaSpecs, t)
			tfJob.Spec.TFReplicaSpecs[typ] = spec
			return
		}
	}
}

// SuccessPolicy is the success policy.
type SuccessPolicy string

const (
	SuccessPolicyDefault    SuccessPolicy = ""
	SuccessPolicyAllWorkers SuccessPolicy = "AllWorkers"
)

// SetDefaults_TFJob sets any unspecified values to defaults.
func SetDefaults_TFJob(tfjob *TFJob) {
	// Set default cleanpod policy to Running.
	if tfjob.Spec.RunPolicy.CleanPodPolicy == nil {
		running := commonv1.CleanPodPolicyRunning
		tfjob.Spec.RunPolicy.CleanPodPolicy = &running
	}
	// Set default success policy to "".
	if tfjob.Spec.SuccessPolicy == nil {
		defaultPolicy := SuccessPolicyDefault
		tfjob.Spec.SuccessPolicy = &defaultPolicy
	}

	// Update the key of TFReplicaSpecs to camel case.
	setTFTypeNamesToCamelCase(tfjob)

	for _, spec := range tfjob.Spec.TFReplicaSpecs {
		// Set default replicas to 1.
		setTFDefaultReplicas(spec)
		// Set default port to tensorFlow container.
		setTFDefaultPort(&spec.Template.Spec)
	}
}

// setDefaultPort sets the default ports for mxnet container.
func setXGBoostDefaultPort(spec *v1.PodSpec) {
	index := 0
	for i, container := range spec.Containers {
		if container.Name == XGBoostDefaultContainerName {
			index = i
			break
		}
	}

	hasMXJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == XGBoostDefaultPortName {
			hasMXJobPort = true
			break
		}
	}
	if !hasMXJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          XGBoostDefaultPortName,
			ContainerPort: XGBoostDefaultPort,
		})
	}
}

func setXGBoostDefaultReplicas(spec *commonv1.ReplicaSpec) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = XGBoostDefaultRestartPolicy
	}
}

// setTypeNamesToCamelCase sets the name of all replica types from any case to correct case.
func setXGBoostTypeNamesToCamelCase(xgboostJob *XGBoostJob) {
	setXGBoostTypeNameToCamelCase(xgboostJob, XGBoostReplicaTypeMaster)
	setXGBoostTypeNameToCamelCase(xgboostJob, XGBoostReplicaTypeWorker)

}

// setTypeNameToCamelCase sets the name of the replica type from any case to correct case.
// E.g. from server to Server; from WORKER to Worker.
func setXGBoostTypeNameToCamelCase(xgboostJob *XGBoostJob, typ commonv1.ReplicaType) {
	for t := range xgboostJob.Spec.XGBReplicaSpecs {
		if strings.EqualFold(string(t), string(typ)) && t != typ {
			spec := xgboostJob.Spec.XGBReplicaSpecs[t]
			delete(xgboostJob.Spec.XGBReplicaSpecs, t)
			xgboostJob.Spec.XGBReplicaSpecs[typ] = spec
			return
		}
	}
}

// SetDefaults_XGBoostJob sets any unspecified values to defaults.
func SetDefaults_XGBoostJob(xgboostJob *XGBoostJob) {
	// Set default cleanpod policy to All.
	if xgboostJob.Spec.RunPolicy.CleanPodPolicy == nil {
		all := commonv1.CleanPodPolicyAll
		xgboostJob.Spec.RunPolicy.CleanPodPolicy = &all
	}

	// Update the key of MXReplicaSpecs to camel case.
	setXGBoostTypeNamesToCamelCase(xgboostJob)

	for _, spec := range xgboostJob.Spec.XGBReplicaSpecs {
		// Set default replicas to 1.
		setXGBoostDefaultReplicas(spec)
		// Set default port to mxnet container.
		setXGBoostDefaultPort(&spec.Template.Spec)
	}
}
