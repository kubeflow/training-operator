package jobcontroller

import (
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/kubeflow/tf-operator/pkg/control"
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

		if _, ok := service.Labels[jc.Controller.GetReplicaTypeLabelKey()]; !ok {
			log.Infof("This service maybe not created by %v", jc.Controller.ControllerName())
			return
		}

		rtype := service.Labels[jc.Controller.GetReplicaTypeLabelKey()]
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

// getServicesForJob returns the set of services that this job should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned services are pointers into the cache.
func (jc *JobController) GetServicesForJob(job metav1.Object) ([]*v1.Service, error) {
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
	var result []*v1.Service

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[jc.Controller.GetReplicaTypeLabelKey()] = replicaType

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
	serviceSlices := make([][]*v1.Service, replicas)
	for _, service := range services {
		if _, ok := service.Labels[jc.Controller.GetReplicaIndexLabelKey()]; !ok {
			logger.Warning("The service do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(service.Labels[jc.Controller.GetReplicaIndexLabelKey()])
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
