/*
Copyright 2016 The Kubernetes Authors.

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

package controller

import (
	"fmt"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/kubernetes/pkg/controller"
)

type ServiceControllerRefManager struct {
	controller.BaseControllerRefManager

	controllerKind schema.GroupVersionKind
	serviceControl ServiceControlInterface
}

// NewServiceControllerRefManager returns a ServiceControllerRefManager that exposes
// methods to manage the controllerRef of services.
//
// The canAdopt() function can be used to perform a potentially expensive check
// (such as a live GET from the API server) prior to the first adoption.
// It will only be called (at most once) if an adoption is actually attempted.
// If canAdopt() returns a non-nil error, all adoptions will fail.
//
// NOTE: Once canAdopt() is called, it will not be called again by the same
//       ServiceControllerRefManager instance. Create a new instance if it makes
//       sense to check canAdopt() again (e.g. in a different sync pass).
func NewServiceControllerRefManager(
	serviceControl ServiceControlInterface,
	ctr metav1.Object,
	selector labels.Selector,
	controllerKind schema.GroupVersionKind,
	canAdopt func() error,
) *ServiceControllerRefManager {
	return &ServiceControllerRefManager{
		BaseControllerRefManager: controller.BaseControllerRefManager{
			Controller:   ctr,
			Selector:     selector,
			CanAdoptFunc: canAdopt,
		},
		controllerKind: controllerKind,
		serviceControl: serviceControl,
	}
}

// ClaimServices tries to take ownership of a list of Services.
//
// It will reconcile the following:
//   * Adopt orphans if the selector matches.
//   * Release owned objects if the selector no longer matches.
//
// Optional: If one or more filters are specified, a Service will only be claimed if
// all filters return true.
//
// A non-nil error is returned if some form of reconciliation was attempted and
// failed. Usually, controllers should try again later in case reconciliation
// is still needed.
//
// If the error is nil, either the reconciliation succeeded, or no
// reconciliation was necessary. The list of Services that you now own is returned.
func (m *ServiceControllerRefManager) ClaimServices(services []*v1.Service, filters ...func(*v1.Service) bool) ([]*v1.Service, error) {
	var claimed []*v1.Service
	var errlist []error

	match := func(obj metav1.Object) bool {
		service := obj.(*v1.Service)
		// Check selector first so filters only run on potentially matching Services.
		if !m.Selector.Matches(labels.Set(service.Labels)) {
			return false
		}
		for _, filter := range filters {
			if !filter(service) {
				return false
			}
		}
		return true
	}
	adopt := func(obj metav1.Object) error {
		return m.AdoptService(obj.(*v1.Service))
	}
	release := func(obj metav1.Object) error {
		return m.ReleaseService(obj.(*v1.Service))
	}

	for _, service := range services {
		ok, err := m.ClaimObject(service, match, adopt, release)
		if err != nil {
			errlist = append(errlist, err)
			continue
		}
		if ok {
			claimed = append(claimed, service)
		}
	}
	return claimed, utilerrors.NewAggregate(errlist)
}

// AdoptService sends a patch to take control of the service. It returns the error if
// the patching fails.
func (m *ServiceControllerRefManager) AdoptService(service *v1.Service) error {
	if err := m.CanAdopt(); err != nil {
		return fmt.Errorf("can't adopt Service %v/%v (%v): %v", service.Namespace, service.Name, service.UID, err)
	}
	// Note that ValidateOwnerReferences() will reject this patch if another
	// OwnerReference exists with controller=true.
	addControllerPatch := fmt.Sprintf(
		`{"metadata":{"ownerReferences":[{"apiVersion":"%s","kind":"%s","name":"%s","uid":"%s","controller":true,"blockOwnerDeletion":true}],"uid":"%s"}}`,
		m.controllerKind.GroupVersion(), m.controllerKind.Kind,
		m.Controller.GetName(), m.Controller.GetUID(), service.UID)
	return m.serviceControl.PatchService(service.Namespace, service.Name, []byte(addControllerPatch))
}

// ReleaseService sends a patch to free the service from the control of the controller.
// It returns the error if the patching fails. 404 and 422 errors are ignored.
func (m *ServiceControllerRefManager) ReleaseService(service *v1.Service) error {
	glog.V(2).Infof("patching service %s_%s to remove its controllerRef to %s/%s:%s",
		service.Namespace, service.Name, m.controllerKind.GroupVersion(), m.controllerKind.Kind, m.Controller.GetName())
	deleteOwnerRefPatch := fmt.Sprintf(`{"metadata":{"ownerReferences":[{"$patch":"delete","uid":"%s"}],"uid":"%s"}}`, m.Controller.GetUID(), service.UID)
	err := m.serviceControl.PatchService(service.Namespace, service.Name, []byte(deleteOwnerRefPatch))
	if err != nil {
		if errors.IsNotFound(err) {
			// If the service no longer exists, ignore it.
			return nil
		}
		if errors.IsInvalid(err) {
			// Invalid error will be returned in two cases: 1. the service
			// has no owner reference, 2. the uid of the service doesn't
			// match, which means the service is deleted and then recreated.
			// In both cases, the error can be ignored.

			// TODO: If the service has owner references, but none of them
			// has the owner.UID, server will silently ignore the patch.
			// Investigate why.
			return nil
		}
	}
	return err
}

// RecheckDeletionTimestamp returns a CanAdopt() function to recheck deletion.
//
// The CanAdopt() function calls getObject() to fetch the latest value,
// and denies adoption attempts if that object has a non-nil DeletionTimestamp.
func RecheckDeletionTimestamp(getObject func() (metav1.Object, error)) func() error {
	return func() error {
		obj, err := getObject()
		if err != nil {
			return fmt.Errorf("can't recheck DeletionTimestamp: %v", err)
		}
		if obj.GetDeletionTimestamp() != nil {
			return fmt.Errorf("%v/%v has just been deleted at %v", obj.GetNamespace(), obj.GetName(), obj.GetDeletionTimestamp())
		}
		return nil
	}
}
