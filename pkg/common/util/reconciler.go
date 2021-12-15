// Copyright 2021 The Kubeflow Authors
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
// limitations under the License

package util

import (
	"fmt"
	"reflect"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	commonutil "github.com/kubeflow/common/pkg/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// SatisfiedExpectations returns true if the required adds/dels for the given mxjob have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func SatisfiedExpectations(exp expectation.ControllerExpectationsInterface, jobKey string, replicaTypes []commonv1.ReplicaType) bool {
	satisfied := false
	for _, rtype := range replicaTypes {
		// Check the expectations of the pods.
		expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, string(rtype))
		satisfied = satisfied || exp.SatisfiedExpectations(expectationPodsKey)
		// Check the expectations of the services.
		expectationServicesKey := expectation.GenExpectationServicesKey(jobKey, string(rtype))
		satisfied = satisfied || exp.SatisfiedExpectations(expectationServicesKey)
	}

	return satisfied
}

// OnDependentCreateFunc modify expectations when dependent (pod/service) creation observed.
func OnDependentCreateFunc(exp expectation.ControllerExpectationsInterface) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		rtype := e.Object.GetLabels()[commonv1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		//logrus.Info("Update on create function ", ptjr.ControllerName(), " create object ", e.Object.GetName())
		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			jobKey := fmt.Sprintf("%s/%s", e.Object.GetNamespace(), controllerRef.Name)
			var expectKey string
			if _, ok := e.Object.(*corev1.Pod); ok {
				expectKey = expectation.GenExpectationPodsKey(jobKey, rtype)
			}

			if _, ok := e.Object.(*corev1.Service); ok {
				expectKey = expectation.GenExpectationServicesKey(jobKey, rtype)
			}
			exp.CreationObserved(expectKey)
			return true
		}

		return true
	}
}

// OnDependentUpdateFunc modify expectations when dependent (pod/service) update observed.
func OnDependentUpdateFunc(jc *common.JobController) func(updateEvent event.UpdateEvent) bool {
	return func(e event.UpdateEvent) bool {
		newObj := e.ObjectNew
		oldObj := e.ObjectOld
		if newObj.GetResourceVersion() == oldObj.GetResourceVersion() {
			// Periodic resync will send update events for all known pods.
			// Two different versions of the same pod will always have different RVs.
			return false
		}

		var logger *logrus.Entry
		if _, ok := newObj.(*corev1.Pod); ok {
			logger = commonutil.LoggerForPod(newObj.(*corev1.Pod), jc.Controller.GetAPIGroupVersionKind().Kind)
		}

		if _, ok := newObj.(*corev1.Service); ok {
			logger = commonutil.LoggerForService(newObj.(*corev1.Service), jc.Controller.GetAPIGroupVersionKind().Kind)
		}

		newControllerRef := metav1.GetControllerOf(newObj)
		oldControllerRef := metav1.GetControllerOf(oldObj)
		controllerRefChanged := !reflect.DeepEqual(newControllerRef, oldControllerRef)

		if controllerRefChanged && oldControllerRef != nil {
			// The ControllerRef was changed. Sync the old controller, if any.
			if job := resolveControllerRef(jc, oldObj.GetName(), oldControllerRef); job != nil {
				logger.Infof("pod/service controller ref updated: %v, %v", newObj, oldObj)
				return true
			}
		}

		// If it has a controller ref, that's all that matters.
		if newControllerRef != nil {
			job := resolveControllerRef(jc, newObj.GetNamespace(), newControllerRef)
			if job == nil {
				return false
			}
			logger.Debugf("pod/service has a controller ref: %v, %v", newObj, oldObj)
			return true
		}
		return false
	}
}

// resolveControllerRef returns the job referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching job
// of the correct Kind.
func resolveControllerRef(jc *common.JobController, namespace string, controllerRef *metav1.OwnerReference) metav1.Object {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != jc.Controller.GetAPIGroupVersionKind().Kind {
		return nil
	}
	job, err := jc.Controller.GetJobFromInformerCache(namespace, controllerRef.Name)
	if err != nil {
		return nil
	}
	if job.GetUID() != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return job
}

// OnDependentDeleteFunc modify expectations when dependent (pod/service) deletion observed.
func OnDependentDeleteFunc(exp expectation.ControllerExpectationsInterface) func(event.DeleteEvent) bool {
	return func(e event.DeleteEvent) bool {

		rtype := e.Object.GetLabels()[commonv1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		// logrus.Info("Update on deleting function ", xgbr.ControllerName(), " delete object ", e.Object.GetName())
		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			jobKey := fmt.Sprintf("%s/%s", e.Object.GetNamespace(), controllerRef.Name)
			var expectKey string
			if _, ok := e.Object.(*corev1.Pod); ok {
				expectKey = expectation.GenExpectationPodsKey(jobKey, rtype)
			}

			if _, ok := e.Object.(*corev1.Service); ok {
				expectKey = expectation.GenExpectationServicesKey(jobKey, rtype)
			}

			exp.DeletionObserved(expectKey)
			return true
		}

		return true
	}
}
