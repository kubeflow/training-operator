/*
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

package xgboost

import (
	"fmt"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	v1 "github.com/kubeflow/tf-operator/pkg/apis/xgboost/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// satisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (r *XGBoostJobReconciler) satisfiedExpectations(xgbJob *v1.XGBoostJob) bool {
	satisfied := false
	key, err := common.KeyFunc(xgbJob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", xgbJob, err))
		return false
	}
	for rtype := range xgbJob.Spec.XGBReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := expectation.GenExpectationPodsKey(key, string(rtype))
		satisfied = satisfied || r.Expectations.SatisfiedExpectations(expectationPodsKey)
		// Check the expectations of the services.
		expectationServicesKey := expectation.GenExpectationServicesKey(key, string(rtype))
		satisfied = satisfied || r.Expectations.SatisfiedExpectations(expectationServicesKey)
	}
	return satisfied
}

// onDependentCreateFunc modify expectations when dependent (pod/service) creation observed.
func onDependentCreateFunc(r reconcile.Reconciler) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		xgbr, ok := r.(*XGBoostJobReconciler)
		if !ok {
			return true
		}
		rtype := e.Object.GetLabels()[commonv1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		logrus.Info("Update on create function ", xgbr.ControllerName(), " create object ", e.Object.GetName())
		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			var expectKey string
			if _, ok := e.Object.(*corev1.Pod); ok {
				expectKey = expectation.GenExpectationPodsKey(e.Object.GetNamespace()+"/"+controllerRef.Name, rtype)
			}

			if _, ok := e.Object.(*corev1.Service); ok {
				expectKey = expectation.GenExpectationServicesKey(e.Object.GetNamespace()+"/"+controllerRef.Name, rtype)
			}
			xgbr.Expectations.CreationObserved(expectKey)
			return true
		}

		return true
	}
}

// onDependentDeleteFunc modify expectations when dependent (pod/service) deletion observed.
func onDependentDeleteFunc(r reconcile.Reconciler) func(event.DeleteEvent) bool {
	return func(e event.DeleteEvent) bool {
		xgbr, ok := r.(*XGBoostJobReconciler)
		if !ok {
			return true
		}

		rtype := e.Object.GetLabels()[commonv1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		logrus.Info("Update on deleting function ", xgbr.ControllerName(), " delete object ", e.Object.GetName())
		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			var expectKey string
			if _, ok := e.Object.(*corev1.Pod); ok {
				expectKey = expectation.GenExpectationPodsKey(e.Object.GetNamespace()+"/"+controllerRef.Name, rtype)
			}

			if _, ok := e.Object.(*corev1.Service); ok {
				expectKey = expectation.GenExpectationServicesKey(e.Object.GetNamespace()+"/"+controllerRef.Name, rtype)
			}

			xgbr.Expectations.DeletionObserved(expectKey)
			return true
		}

		return true
	}
}
