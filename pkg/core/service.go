/*
Copyright 2023 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	"fmt"

	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	utillabels "github.com/kubeflow/training-operator/pkg/util/labels"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// FilterServicesForReplicaType returns service belong to a replicaType.
func FilterServicesForReplicaType(services []*v1.Service, replicaType string) ([]*v1.Service, error) {
	var result []*v1.Service

	selector := labels.SelectorFromValidatedSet(labels.Set{
		apiv1.ReplicaTypeLabel: replicaType,
	})

	for _, service := range services {
		set := labels.Set(service.Labels)
		if !selector.Matches(set) {
			continue
		}
		result = append(result, service)
	}
	return result, nil
}

// GetServiceSlices returns a slice, which element is the slice of service.
// Assume the return object is serviceSlices, then serviceSlices[i] is an
// array of pointers to services corresponding to Services for replica i.
func GetServiceSlices(services []*v1.Service, replicas int, logger *log.Entry) [][]*v1.Service {
	serviceSlices := make([][]*v1.Service, CalculateServiceSliceSize(services, replicas))
	for _, service := range services {
		index, err := utillabels.ReplicaIndex(service.Labels)
		if err != nil {
			logger.Warningf("Error obtaining index for service %s/%s: %v", service.Namespace, service.Name, err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d, service: %s/%s", index, service.Namespace, service.Name)
		}

		serviceSlices[index] = append(serviceSlices[index], service)
	}
	return serviceSlices
}

// CalculateServiceSliceSize compare max pod index with desired replicas and return larger size
func CalculateServiceSliceSize(services []*v1.Service, replicas int) int {
	size := 0
	for _, svc := range services {
		index, err := utillabels.ReplicaIndex(svc.Labels)
		if err != nil {
			continue
		}
		size = MaxInt(size, index)
	}

	// size comes from index, need to +1 to indicate real size
	return MaxInt(size+1, replicas)
}

// GetPortsFromJob gets the ports of job container. Port could be nil, if distributed communication strategy doesn't need and no other ports that need to be exposed.
func GetPortsFromJob(spec *apiv1.ReplicaSpec, defaultContainerName string) (map[string]int32, error) {
	ports := make(map[string]int32)

	containers := spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Name == defaultContainerName {
			containerPorts := container.Ports
			if len(containerPorts) == 0 {
				return nil, nil
			}
			for _, port := range containerPorts {
				ports[port.Name] = port.ContainerPort
			}
			return ports, nil
		}
	}

	return nil, fmt.Errorf("failed to find the port")
}
