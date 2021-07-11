package mxnet

import (
	"fmt"
	commonutil "github.com/kubeflow/common/pkg/util"
	mxjobv1 "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	"github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var defaultCleanPodPolicy = commonv1.CleanPodPolicyNone

// satisfiedExpectations returns true if the required adds/dels for the given mxjob have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.

func (r *MXJobReconciler) satisfiedExpectations(job *mxjobv1.MXJob) bool {
	satisfied := false
	key, err := common.KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return false
	}

	for rtype := range job.Spec.MXReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := expectation.GenExpectationPodsKey(key, string(rtype))
		satisfied = satisfied || r.Expectations.SatisfiedExpectations(expectationPodsKey)
		// Check the expectations of the services.
		expectationServicesKey := expectation.GenExpectationServicesKey(key, string(rtype))
		satisfied = satisfied || r.Expectations.SatisfiedExpectations(expectationServicesKey)
	}

	return satisfied
}

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc(r reconcile.Reconciler) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		mxjob, ok := e.Object.(*mxjobv1.MXJob)
		if !ok {
			return true
		}

		// TODO: check default setting
		scheme.Scheme.Default(mxjob)
		msg := fmt.Sprintf("xgboostJob %s is created.", e.Object.GetName())
		logrus.Info(msg)
		//specific the run policy

		if mxjob.Spec.RunPolicy.CleanPodPolicy == nil {
			mxjob.Spec.RunPolicy.CleanPodPolicy = new(commonv1.CleanPodPolicy)
			mxjob.Spec.RunPolicy.CleanPodPolicy = &defaultCleanPodPolicy
		}

		if err := commonutil.UpdateJobConditions(&mxjob.Status, commonv1.JobCreated, "MXJobCreated", msg); err != nil {
			logrus.Error(err, "append job condition error")
			return false
		}
		return true
	}
}

// onDependentCreateFunc modify expectations when dependent (pod/service) creation observed.
func onDependentCreateFunc(r reconcile.Reconciler) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		ptjr, ok := r.(*MXJobReconciler)
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
		xgbr, ok := r.(*MXJobReconciler)
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
