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
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/controller.v1/common"
	"github.com/kubeflow/training-operator/pkg/controller.v1/expectation"
)

// GenExpectationGenericKey generates an expectation key for {Kind} of a job
func GenExpectationGenericKey(jobKey string, replicaType string, pl string) string {
	return jobKey + "/" + strings.ToLower(replicaType) + "/" + pl
}

// LoggerForGenericKind generates log entry for generic Kubernetes resource Kind
func LoggerForGenericKind(obj metav1.Object, kind string) *log.Entry {
	job := ""
	if controllerRef := metav1.GetControllerOf(obj); controllerRef != nil {
		if controllerRef.Kind == kind {
			job = obj.GetNamespace() + "." + controllerRef.Name
		}
	}
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job": job,
		kind:  obj.GetNamespace() + "." + obj.GetName(),
		"uid": obj.GetUID(),
	})
}

func objectKind(s *runtime.Scheme, obj client.Object) schema.GroupVersionKind {
	gkvs, _, err := s.ObjectKinds(obj)
	if err != nil {
		var logger = LoggerForGenericKind(obj, "")
		logger.Errorf("unknown kind for %v", obj)
		return schema.GroupVersionKind{}
	}
	return gkvs[0]
}

func OnDependentFuncs[T client.Object](s *runtime.Scheme, expectations expectation.ControllerExpectationsInterface, jobController *common.JobController) predicate.TypedFuncs[T] {
	return predicate.TypedFuncs[T]{
		CreateFunc: OnDependentCreateFuncGeneric[T](s, expectations),
		UpdateFunc: OnDependentUpdateFuncGeneric[T](s, jobController),
		DeleteFunc: OnDependentDeleteFuncGeneric[T](s, expectations),
	}
}

// OnDependentCreateFuncGeneric modify expectations when dependent (pod/service) creation observed.
func OnDependentCreateFuncGeneric[T client.Object](s *runtime.Scheme, exp expectation.ControllerExpectationsInterface) func(createEvent event.TypedCreateEvent[T]) bool {
	return func(e event.TypedCreateEvent[T]) bool {
		rtype := e.Object.GetLabels()[kubeflowv1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			jobKey := fmt.Sprintf("%s/%s", e.Object.GetNamespace(), controllerRef.Name)
			kind := e.Object.GetObjectKind().GroupVersionKind().Kind
			if kind == "" {
				kind = objectKind(s, e.Object).Kind
			}
			pl := strings.ToLower(kind) + "s"
			expectKey := GenExpectationGenericKey(jobKey, rtype, pl)
			exp.CreationObserved(expectKey)
			return true
		}

		return true
	}
}

// OnDependentUpdateFuncGeneric modify expectations when dependent update observed.
func OnDependentUpdateFuncGeneric[T client.Object](_ *runtime.Scheme, jc *common.JobController) func(updateEvent event.TypedUpdateEvent[T]) bool {
	return func(e event.TypedUpdateEvent[T]) bool {
		newObj := e.ObjectNew
		oldObj := e.ObjectOld
		if newObj.GetResourceVersion() == oldObj.GetResourceVersion() {
			// Periodic resync will send update events for all known pods.
			// Two different versions of the same pod will always have different RVs.
			return false
		}

		kind := jc.Controller.GetAPIGroupVersionKind().Kind
		var logger = LoggerForGenericKind(newObj, kind)

		newControllerRef := metav1.GetControllerOf(newObj)
		oldControllerRef := metav1.GetControllerOf(oldObj)
		controllerRefChanged := !reflect.DeepEqual(newControllerRef, oldControllerRef)

		if controllerRefChanged && oldControllerRef != nil {
			// The ControllerRef was changed. Sync the old controller, if any.
			if job := resolveControllerRef(jc, oldObj.GetNamespace(), oldControllerRef); job != nil {
				logger.Infof("%s controller ref updated: %v, %v", kind, newObj, oldObj)
				return true
			}
		}

		// If it has a controller ref, that's all that matters.
		if newControllerRef != nil {
			job := resolveControllerRef(jc, newObj.GetNamespace(), newControllerRef)
			if job == nil {
				return false
			}
			logger.Debugf("%s has a controller ref: %v, %v", kind, newObj, oldObj)
			return true
		}
		return false
	}
}

// OnDependentDeleteFuncGeneric modify expectations when dependent deletion observed.
func OnDependentDeleteFuncGeneric[T client.Object](s *runtime.Scheme, exp expectation.ControllerExpectationsInterface) func(event.TypedDeleteEvent[T]) bool {
	return func(e event.TypedDeleteEvent[T]) bool {
		rtype := e.Object.GetLabels()[kubeflowv1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			jobKey := fmt.Sprintf("%s/%s", e.Object.GetNamespace(), controllerRef.Name)
			kind := e.Object.GetObjectKind().GroupVersionKind().Kind
			if kind == "" {
				kind = objectKind(s, e.Object).Kind
			}
			pl := strings.ToLower(kind) + "s"
			expectKey := GenExpectationGenericKey(jobKey, rtype, pl)
			exp.DeletionObserved(expectKey)
			return true
		}

		return true
	}
}

// SatisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func SatisfiedExpectations(exp expectation.ControllerExpectationsInterface, jobKey string, replicaTypes []kubeflowv1.ReplicaType) bool {
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
