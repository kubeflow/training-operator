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

// Package controller provides a Kubernetes controller for a TFJob resource.
package controller

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/control"
	"github.com/kubeflow/tf-operator/pkg/generator"
)

// reconcileServices checks and updates services for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting services.
func (tc *TFJobController) reconcileServices(
	tfjob *tfv1alpha2.TFJob,
	services []*v1.Service,
	rtype tfv1alpha2.TFReplicaType,
	spec *tfv1alpha2.TFReplicaSpec) error {

	// Convert TFReplicaType to lower string.
	rt := strings.ToLower(string(rtype))

	replicas := int(*spec.Replicas)
	// Get all services for the type rt.
	services = filterServicesForTFReplicaType(services, rt)

	serviceSlices := getServiceSlices(services, replicas, loggerForReplica(tfjob, rt))

	for index, serviceSlice := range serviceSlices {
		if len(serviceSlice) > 1 {
			loggerForReplica(tfjob, rt).Warningf("We have too many services for %s %d", rt, index)
			// TODO(gaocegege): Kill some services.
		} else if len(serviceSlice) == 0 {
			loggerForReplica(tfjob, rt).Infof("need to create new service: %s-%d", rt, index)
			err := tc.createNewService(tfjob, rtype, strconv.Itoa(index), spec)
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
		if _, ok := service.Labels[tfReplicaIndexLabel]; !ok {
			logger.Warning("The service do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(service.Labels[tfReplicaIndexLabel])
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
func (tc *TFJobController) createNewService(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType, index string, spec *tfv1alpha2.TFReplicaSpec) error {
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for tfjob object %#v: %v", tfjob, err))
		return err
	}

	// Convert TFReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	expectationServicesKey := genExpectationServicesKey(tfjobKey, rt)
	err = tc.expectations.ExpectCreations(expectationServicesKey, 1)
	if err != nil {
		return err
	}

	// Create OwnerReference.
	controllerRef := generator.GenOwnerReference(tfjob)

	// Append tfReplicaTypeLabel and tfReplicaIndexLabel labels.
	labels := generator.GenLabels(tfjob.Name)
	labels[tfReplicaTypeLabel] = rt
	labels[tfReplicaIndexLabel] = index

	port, err := generator.GetPortFromTFJob(tfjob, rtype)
	if err != nil {
		return err
	}

	service := &v1.Service{
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []v1.ServicePort{
				{
					Name: tfv1alpha2.DefaultPortName,
					Port: port,
				},
			},
		},
	}

	service.Name = generator.GenGeneralName(tfjob.Name, rt, index)
	service.Labels = labels

	err = tc.serviceControl.CreateServicesWithControllerRef(tfjob.Namespace, service, tfjob, controllerRef)
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

// getServicesForTFJob returns the set of services that this tfjob should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned services are pointers into the cache.
func (tc *TFJobController) getServicesForTFJob(tfjob *tfv1alpha2.TFJob) ([]*v1.Service, error) {
	// Create selector
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: generator.GenLabels(tfjob.Name),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all services to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	services, err := tc.serviceLister.Services(tfjob.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing services (see #42639).
	canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := tc.tfJobClientSet.KubeflowV1alpha2().TFJobs(tfjob.Namespace).Get(tfjob.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.UID != tfjob.UID {
			return nil, fmt.Errorf("original TFJob %v/%v is gone: got uid %v, wanted %v", tfjob.Namespace, tfjob.Name, fresh.UID, tfjob.UID)
		}
		return fresh, nil
	})
	cm := control.NewServiceControllerRefManager(tc.serviceControl, tfjob, selector, controllerKind, canAdoptFunc)
	return cm.ClaimServices(services)
}

// filterServicesForTFReplicaType returns service belong to a TFReplicaType.
func filterServicesForTFReplicaType(services []*v1.Service, tfReplicaType string) []*v1.Service {
	var result []*v1.Service

	tfReplicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	tfReplicaSelector.MatchLabels[tfReplicaTypeLabel] = tfReplicaType

	for _, service := range services {
		selector, _ := metav1.LabelSelectorAsSelector(tfReplicaSelector)
		if !selector.Matches(labels.Set(service.Labels)) {
			continue
		}
		result = append(result, service)
	}
	return result
}

func genExpectationServicesKey(tfjobKey, replicaType string) string {
	return tfjobKey + "/" + strings.ToLower(replicaType) + "/services"
}

// When a service is created, enqueue the controller that manages it and update its expectations.
func (tc *TFJobController) addService(obj interface{}) {
	service := obj.(*v1.Service)
	if service.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new service shows up in a state that
		// is already pending deletion. Prevent the service from being a creation observation.
		// tc.deleteService(service)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(service); controllerRef != nil {
		tfjob := tc.resolveControllerRef(service.Namespace, controllerRef)
		if tfjob == nil {
			return
		}

		tfjobKey, err := KeyFunc(tfjob)
		if err != nil {
			return
		}

		if _, ok := service.Labels[tfReplicaTypeLabel]; !ok {
			log.Infof("This service maybe not created by tf-operator")
			return
		}

		rtype := service.Labels[tfReplicaTypeLabel]
		expectationServicesKey := genExpectationServicesKey(tfjobKey, rtype)

		tc.expectations.CreationObserved(expectationServicesKey)
		tc.enqueueTFJob(tfjob)

		return
	}

}

// When a service is updated, figure out what tfjob/s manage it and wake them up.
// If the labels of the service have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.Service types.
func (tc *TFJobController) updateService(old, cur interface{}) {
	// TODO(CPH): handle this gracefully.
}

// When a service is deleted, enqueue the tfjob that manages the service and update its expectations.
// obj could be an *v1.Service, or a DeletionFinalStateUnknown marker item.
func (tc *TFJobController) deleteService(obj interface{}) {
	// TODO(CPH): handle this gracefully.
}
