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

package pytorch

import (
	"fmt"

	v1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1alpha2"
)

var (
	errPortNotFound = fmt.Errorf("Failed to found the port")
)

// GetPortFromPyTorchJob gets the port of pytorch container.
func GetPortFromPyTorchJob(job *v1alpha2.PyTorchJob, rtype v1alpha2.PyTorchReplicaType) (int32, error) {
	containers := job.Spec.PyTorchReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == v1alpha2.DefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == v1alpha2.DefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, errPortNotFound
}

func ContainMasterSpec(job *v1alpha2.PyTorchJob) bool {
	if _, ok := job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster]; ok {
		return true
	}
	return false
}
