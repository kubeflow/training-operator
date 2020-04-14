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
package common

import (
	"fmt"
	"strconv"
	"strings"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/controller"
)

// When a service is created, enqueue the controller that manages it and update its expectations.
func (jc *JobController) AddService(obj interface{}) {
	service := obj.(*v1.Service)
	if service.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new service shows up in a state that
		// is already pending deletion. Prevent the service from being a creation observation.
		// tc.deleteService(service)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(service); controllerRef != nil {
		job := jc.resolveControllerRef(service.Namespace, controllerRef)
		if job == nil {
			return
		}

		jobKey, err := controller.KeyFunc(job)
		if err != nil {
			return
		}

		if _, ok := service.Labels[apiv1.ReplicaTypeLabel]; !ok {
			log.Infof("This service maybe not created by %v", jc.Controller.ControllerName())
			return
		}

		rtype := service.Labels[apiv1.ReplicaTypeLabel]
		expectationServicesKey := GenExpectationServicesKey(jobKey, rtype)

		jc.Expectations.CreationObserved(expectationServicesKey)
		// TODO: we may need add backoff here
		jc.WorkQueue.Add(jobKey)

		return
	}

}

// When a service is updated, figure out what job/s manage it and wake them up.
// If the labels of the service have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.Service types.
func (jc *JobController) UpdateService(old, cur interface{}) {
	// TODO(CPH): handle this gracefully.
}

// When a service is deleted, enqueue the job that manages the service and update its expectations.
// obj could be an *v1.Service, or a DeletionFinalStateUnknown marker item.
func (jc *JobController) DeleteService(obj interface{}) {
	// TODO(CPH): handle this gracefully.
}

// FilterServicesForReplicaType returns service belong to a replicaType.
func (jc *JobController) FilterServicesForReplicaType(services []*v1.Service, replicaType string) ([]*v1.Service, error) {
	var result []*v1.Service

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[apiv1.ReplicaTypeLabel] = replicaType

	for _, service := range services {
		selector, err := metav1.LabelSelectorAsSelector(replicaSelector)
		if err != nil {
			return nil, err
		}
		if !selector.Matches(labels.Set(service.Labels)) {
			continue
		}
		result = append(result, service)
	}
	return result, nil
}

// getServiceSlices returns a slice, which element is the slice of service.
// Assume the return object is serviceSlices, then serviceSlices[i] is an
// array of pointers to services corresponding to Services for replica i.
func (jc *JobController) GetServiceSlices(services []*v1.Service, replicas int, logger *log.Entry) [][]*v1.Service {
	serviceSlices := make([][]*v1.Service, calculateServiceSliceSize(services, replicas))
	for _, service := range services {
		if _, ok := service.Labels[apiv1.ReplicaIndexLabel]; !ok {
			logger.Warning("The service do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(service.Labels[apiv1.ReplicaIndexLabel])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d, service: %s/%s", index, service.Namespace, service.Name)
		}

		serviceSlices[index] = append(serviceSlices[index], service)
	}
	return serviceSlices
}

// calculateServiceSliceSize compare max pod index with desired replicas and return larger size
func calculateServiceSliceSize(services []*v1.Service, replicas int) int {
	size := 0
	for _, svc := range services {
		if _, ok := svc.Labels[apiv1.ReplicaIndexLabel]; !ok {
			continue
		}
		index, err := strconv.Atoi(svc.Labels[apiv1.ReplicaIndexLabel])
		if err != nil {
			continue
		}
		size = MaxInt(size, index)
	}

	// size comes from index, need to +1 to indicate real size
	return MaxInt(size+1, replicas)
}

// reconcileServices checks and updates services for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting services.
func (jc *JobController) ReconcileServices(
	job metav1.Object,
	services []*v1.Service,
	rtype apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec) error {

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))

	replicas := int(*spec.Replicas)
	// Get all services for the type rt.
	services, err := jc.FilterServicesForReplicaType(services, rt)
	if err != nil {
		return err
	}

	// GetServiceSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have services with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a svc with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], svc with replica-index 1 and 2 are out of range and will be deleted.
	serviceSlices := jc.GetServiceSlices(services, replicas, commonutil.LoggerForReplica(job, rt))

	for index, serviceSlice := range serviceSlices {
		if len(serviceSlice) > 1 {
			commonutil.LoggerForReplica(job, rt).Warningf("We have too many services for %s %d", rt, index)
		} else if len(serviceSlice) == 0 {
			commonutil.LoggerForReplica(job, rt).Infof("need to create new service: %s-%d", rt, index)
			err = jc.CreateNewService(job, rtype, spec, strconv.Itoa(index))
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current svc.
			svc := serviceSlice[0]

			// check if the index is in the valid range, if not, we should kill the svc
			if index < 0 || index >= replicas {
				err = jc.Controller.DeleteService(job, svc.Name, svc.Namespace)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// GetPortFromJob gets the port of job container. Port could be nil depending on different distributed communication strategy
func (jc *JobController) GetPortFromJob(spec *apiv1.ReplicaSpec) (*int32, error) {
	// Consider the case controller doesn't use fixed port, headless service without port will enable random pod to pod communication
	if jc.Controller.GetDefaultContainerPortName() == "" {
		return nil, nil
	}

	containers := spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Name == jc.Controller.GetDefaultContainerName() {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == jc.Controller.GetDefaultContainerPortName() {
					return &port.ContainerPort, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("failed to find the port")
}

// createNewService creates a new service for the given index and type.
func (jc *JobController) CreateNewService(job metav1.Object, rtype apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec, index string) error {
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	expectationServicesKey := GenExpectationServicesKey(jobKey, rt)
	err = jc.Expectations.ExpectCreations(expectationServicesKey, 1)
	if err != nil {
		return err
	}

	// Append ReplicaTypeLabel and ReplicaIndexLabel labels.
	labels := jc.GenLabels(job.GetName())
	labels[apiv1.ReplicaTypeLabel] = rt
	labels[apiv1.ReplicaIndexLabel] = index

	port, err := jc.GetPortFromJob(spec)
	if err != nil {
		return err
	}

	service := &v1.Service{
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports:     []v1.ServicePort{},
		},
	}

	// Add service port to headless service only if port is set from controller implementation
	if port != nil {
		svcPort := v1.ServicePort{Name: jc.Controller.GetDefaultContainerPortName(), Port: *port}
		service.Spec.Ports = append(service.Spec.Ports, svcPort)
	}

	service.Name = GenGeneralName(job.GetName(), rt, index)
	service.Labels = labels
	// Create OwnerReference.
	controllerRef := jc.GenOwnerReference(job)

	err = jc.CreateServicesWithControllerRef(job.GetNamespace(), service, job.(runtime.Object), controllerRef)
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

func (jc *JobController) CreateServicesWithControllerRef(namespace string, service *v1.Service, controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return jc.createServices(namespace, service, controllerObject, controllerRef)
}

func (jc *JobController) createServices(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	if labels.Set(service.Labels).AsSelectorPreValidated().Empty() {
		return fmt.Errorf("unable to create Services, no labels")
	}
	serviceWithOwner, err := getServiceFromTemplate(service, object, controllerRef)
	serviceWithOwner.Namespace = namespace
	if err != nil {
		jc.Recorder.Eventf(object, v1.EventTypeWarning, FailedCreateServiceReason, "Error creating: %v", err)
		return fmt.Errorf("unable to create services: %v", err)
	}

	err = jc.Controller.CreateService(object, serviceWithOwner)
	if err != nil {
		jc.Recorder.Eventf(object, v1.EventTypeWarning, FailedCreateServiceReason, "Error creating: %v", err)
		return fmt.Errorf("unable to create services: %v", err)
	}

	accessor, err := meta.Accessor(object)
	if err != nil {
		log.Errorf("parentObject does not have ObjectMeta, %v", err)
		return nil
	}
	log.Infof("Controller %v created service %v", accessor.GetName(), serviceWithOwner.Name)
	jc.Recorder.Eventf(object, v1.EventTypeNormal, SuccessfulCreateServiceReason, "Created service: %v", serviceWithOwner.Name)

	return nil
}
