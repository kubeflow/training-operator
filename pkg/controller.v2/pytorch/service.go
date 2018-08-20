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

// Package controller provides a Kubernetes controller for a PyTorchJob resource.
package pytorch

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	v1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/controller.v2/jobcontroller"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
)

// reconcileServices checks and updates services for each given PyTorchReplicaSpec.
// It will requeue the job in case of an error while creating/deleting services.
func (pc *PyTorchController) reconcileServices(
	job *v1alpha2.PyTorchJob,
	services []*v1.Service,
	rtype v1alpha2.PyTorchReplicaType,
	spec *v1alpha2.PyTorchReplicaSpec) error {

	// Convert PyTorchReplicaType to lower string.
	rt := strings.ToLower(string(rtype))

	replicas := int(*spec.Replicas)
	// Get all services for the type rt.
	services, err := pc.FilterServicesForReplicaType(services, rt)
	if err != nil {
		return err
	}

	serviceSlices := getServiceSlices(services, replicas, pylogger.LoggerForReplica(job, rt))

	for index, serviceSlice := range serviceSlices {
		if len(serviceSlice) > 1 {
			pylogger.LoggerForReplica(job, rt).Warningf("We have too many services for %s %d", rt, index)
			// TODO(gaocegege): Kill some services.
		} else if len(serviceSlice) == 0 {
			pylogger.LoggerForReplica(job, rt).Infof("need to create new service: %s-%d", rt, index)
			err = pc.createNewService(job, rtype, strconv.Itoa(index), spec)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// getServiceSlices returns a slice, which element is the slice of service.
// Assume the return object is serviceSlices, then serviceSlices[i] is an
// array of pointers to services corresponding to Services for replica i.
func getServiceSlices(services []*v1.Service, replicas int, logger *log.Entry) [][]*v1.Service {
	serviceSlices := make([][]*v1.Service, replicas)
	for _, service := range services {
		if _, ok := service.Labels[replicaIndexLabel]; !ok {
			logger.Warning("The service do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(service.Labels[replicaIndexLabel])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d", index)
		} else {
			serviceSlices[index] = append(serviceSlices[index], service)
		}
	}
	return serviceSlices
}

// createNewService creates a new service for the given index and type.
func (pc *PyTorchController) createNewService(job *v1alpha2.PyTorchJob, rtype v1alpha2.PyTorchReplicaType, index string, spec *v1alpha2.PyTorchReplicaSpec) error {
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for job object %#v: %v", job, err))
		return err
	}

	// Convert PyTorchReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	expectationServicesKey := jobcontroller.GenExpectationServicesKey(jobKey, rt)
	err = pc.Expectations.ExpectCreations(expectationServicesKey, 1)
	if err != nil {
		return err
	}

	// Create OwnerReference.
	controllerRef := pc.GenOwnerReference(job)

	// Append replicaTypeLabel and replicaIndexLabel labels.
	labels := pc.GenLabels(job.Name)
	labels[replicaTypeLabel] = rt
	labels[replicaIndexLabel] = index

	port, err := GetPortFromPyTorchJob(job, rtype)
	if err != nil {
		return err
	}

	service := &v1.Service{
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []v1.ServicePort{
				{
					Name: v1alpha2.DefaultPortName,
					Port: port,
				},
			},
		},
	}

	service.Name = jobcontroller.GenGeneralName(job.Name, rt, index)
	service.Labels = labels

	err = pc.ServiceControl.CreateServicesWithControllerRef(job.Namespace, service, job, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Service is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the service keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// service when the expectation expires.
		return nil
	} else if err != nil {
		return err
	}
	return nil
}
