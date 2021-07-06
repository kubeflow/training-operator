package controllers

import (
	"fmt"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	pytorchv1 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *PyTorchJobReconciler) satisfiedExpectations(job *pytorchv1.PyTorchJob) bool {
	satisfied := false
	key, err := common.KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return false
	}

	for rtype := range job.Spec.PyTorchReplicaSpecs {
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
		ptjr, ok := r.(*PyTorchJobReconciler)
		if !ok {
			return true
		}

		rtype := e.Object.GetLabels()[commonv1.ReplicaTypeLabel]
		if len(rtype) == 0 {
			return false
		}

		logrus.Info("Update on create function ", ptjr.ControllerName(), " create object ", e.Object.GetName())
		if controllerRef := metav1.GetControllerOf(e.Object); controllerRef != nil {
			var expectKey string
			if _, ok = e.Object.(*corev1.Pod); ok {
				expectKey = expectation.GenExpectationPodsKey(e.Object.GetNamespace()+"/"+controllerRef.Name, rtype)
			}

			if _, ok = e.Object.(*corev1.Service); ok {
				expectKey = expectation.GenExpectationServicesKey(e.Object.GetNamespace()+"/"+controllerRef.Name, rtype)
			}
			ptjr.Expectations.CreationObserved(expectKey)
			return true
		}

		return true
	}
}

// onDependentDeleteFunc modify expectations when dependent (pod/service) deletion observed.
func onDependentDeleteFunc(r reconcile.Reconciler) func(event.DeleteEvent) bool {
	return func(e event.DeleteEvent) bool {
		xgbr, ok := r.(*PyTorchJobReconciler)
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
			if _, ok = e.Object.(*corev1.Pod); ok {
				expectKey = expectation.GenExpectationPodsKey(e.Object.GetNamespace()+"/"+controllerRef.Name, rtype)
			}

			if _, ok = e.Object.(*corev1.Service); ok {
				expectKey = expectation.GenExpectationServicesKey(e.Object.GetNamespace()+"/"+controllerRef.Name, rtype)
			}

			xgbr.Expectations.DeletionObserved(expectKey)
			return true
		}

		return true
	}
}
