// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/controller.v1/control"
	"github.com/kubeflow/training-operator/pkg/controller.v1/expectation"
	"github.com/kubeflow/training-operator/pkg/core"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
	utillabels "github.com/kubeflow/training-operator/pkg/util/labels"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	succeededServiceCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "succeeded_service_creation_total",
		Help: "The total number of succeeded service creation",
	})
	failedServiceCreationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed_service_creation_total",
		Help: "The total number of failed service creation",
	})
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

		jobKey, err := KeyFunc(job)
		if err != nil {
			return
		}

		rType, err := utillabels.ReplicaType(service.Labels)
		if err != nil {
			log.Infof("This service maybe not created by %v", jc.Controller.ControllerName())
			return
		}

		expectationServicesKey := expectation.GenExpectationServicesKey(jobKey, string(rType))

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

// GetServicesForJob returns the set of services that this job should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned services are pointers into the cache.
func (jc *JobController) GetServicesForJob(jobObject interface{}) ([]*v1.Service, error) {
	job, ok := jobObject.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("job is not of type metav1.Object")
	}

	// Create selector
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: jc.GenLabels(job.GetName()),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all services to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	services, err := jc.ServiceLister.Services(job.GetNamespace()).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing services (see #42639).
	canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := jc.Controller.GetJobFromInformerCache(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := control.NewServiceControllerRefManager(jc.ServiceControl, job, selector, jc.Controller.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimServices(services)
}

// FilterServicesForReplicaType returns service belong to a replicaType.
func (jc *JobController) FilterServicesForReplicaType(services []*v1.Service, replicaType string) ([]*v1.Service, error) {
	return core.FilterServicesForReplicaType(services, replicaType)
}

// GetServiceSlices returns a slice, which element is the slice of service.
// Assume the return object is serviceSlices, then serviceSlices[i] is an
// array of pointers to services corresponding to Services for replica i.
func (jc *JobController) GetServiceSlices(services []*v1.Service, replicas int, logger *log.Entry) [][]*v1.Service {
	return core.GetServiceSlices(services, replicas, logger)
}

// ReconcileServices checks and updates services for each given ReplicaSpec.
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
			commonutil.LoggerForReplica(job, rt).Warningf("We have too many services for %s %d", rtype, index)
		} else if len(serviceSlice) == 0 {
			commonutil.LoggerForReplica(job, rt).Infof("need to create new service: %s-%d", rtype, index)
			err = jc.CreateNewService(job, rtype, spec, strconv.Itoa(index))
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current svc.
			svc := serviceSlice[0]

			// check if the index is in the valid range, if not, we should kill the svc
			if index < 0 || index >= replicas {
				err = jc.ServiceControl.DeleteService(svc.Namespace, svc.Name, job.(runtime.Object))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// GetPortsFromJob gets the ports of job container. Port could be nil, if distributed communication strategy doesn't need and no other ports that need to be exposed.
func (jc *JobController) GetPortsFromJob(spec *apiv1.ReplicaSpec) (map[string]int32, error) {
	return core.GetPortsFromJob(spec, jc.Controller.GetDefaultContainerName())
}

// CreateNewService creates a new service for the given index and type.
func (jc *JobController) CreateNewService(job metav1.Object, rtype apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec, index string) error {
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}

	rt := strings.ToLower(string(rtype))
	labels := jc.GenLabels(job.GetName())
	utillabels.SetReplicaType(labels, rt)
	utillabels.SetReplicaIndexStr(labels, index)

	ports, err := jc.GetPortsFromJob(spec)
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

	// Add service ports to headless service
	for name, port := range ports {
		svcPort := v1.ServicePort{Name: name, Port: port}
		service.Spec.Ports = append(service.Spec.Ports, svcPort)
	}

	service.Name = GenGeneralName(job.GetName(), rt, index)
	service.Labels = labels
	// Create OwnerReference.
	controllerRef := jc.GenOwnerReference(job)

	// Creation is expected when there is no error returned
	expectationServicesKey := expectation.GenExpectationServicesKey(jobKey, rt)
	jc.Expectations.RaiseExpectations(expectationServicesKey, 1, 0)

	err = jc.ServiceControl.CreateServicesWithControllerRef(job.GetNamespace(), service, job.(runtime.Object), controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Service is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the service keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// service when the expectation expires.
		succeededServiceCreationCount.Inc()
		return nil
	} else if err != nil {
		// Since error occurred(the informer won't observe this service),
		// we decrement the expected number of creates
		// and wait until next reconciliation
		jc.Expectations.CreationObserved(expectationServicesKey)
		failedServiceCreationCount.Inc()
		return err
	}
	succeededServiceCreationCount.Inc()
	return nil
}
