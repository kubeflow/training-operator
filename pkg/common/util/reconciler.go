package util

import (
	"fmt"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)




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
