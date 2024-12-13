/*
Copyright 2024 The Kubeflow Authors.

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

package controllerv2

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/kubeflow/training-operator/pkg/constants"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	jobruntimes "github.com/kubeflow/training-operator/pkg/runtime.v2"
)

var errorUnsupportedRuntime = errors.New("the specified runtime is not supported")

type objsOpState int

const (
	creationSucceeded objsOpState = iota
	buildFailed       objsOpState = iota
	creationFailed    objsOpState = iota
	updateFailed      objsOpState = iota
)

type TrainJobReconciler struct {
	log      logr.Logger
	client   client.Client
	recorder record.EventRecorder
	runtimes map[string]jobruntimes.Runtime
}

func NewTrainJobReconciler(client client.Client, recorder record.EventRecorder, runtimes map[string]jobruntimes.Runtime) *TrainJobReconciler {
	return &TrainJobReconciler{
		log:      ctrl.Log.WithName("trainjob-controller"),
		client:   client,
		recorder: recorder,
		runtimes: runtimes,
	}
}

// +kubebuilder:rbac:groups=kubeflow.org,resources=trainjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflow.org,resources=trainjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflow.org,resources=trainjobs/finalizers,verbs=get;update;patch

func (r *TrainJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var trainJob kubeflowv2.TrainJob
	if err := r.client.Get(ctx, req.NamespacedName, &trainJob); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log := ctrl.LoggerFrom(ctx).WithValues("trainJob", klog.KObj(&trainJob))
	ctx = ctrl.LoggerInto(ctx, log)
	log.V(2).Info("Reconciling TrainJob")
	if isTrainJobFinished(&trainJob) {
		log.V(5).Info("TrainJob has already been finished")
		return ctrl.Result{}, nil
	}

	runtimeRefGK := runtimeRefToGroupKind(trainJob.Spec.RuntimeRef).String()
	runtime, ok := r.runtimes[runtimeRefGK]
	if !ok {
		return ctrl.Result{}, fmt.Errorf("%w: %s", errorUnsupportedRuntime, runtimeRefGK)
	}
	opState, err := r.reconcileObjects(ctx, runtime, &trainJob)

	originStatus := trainJob.Status.DeepCopy()
	setSuspendedCondition(&trainJob)
	setCreatedCondition(&trainJob, opState)
	if terminalCondErr := setTerminalCondition(ctx, runtime, &trainJob); terminalCondErr != nil {
		return ctrl.Result{}, errors.Join(err, terminalCondErr)
	}
	if !equality.Semantic.DeepEqual(&trainJob.Status, originStatus) {
		return ctrl.Result{}, errors.Join(err, r.client.Status().Update(ctx, &trainJob))
	}
	return ctrl.Result{}, err
}

func (r *TrainJobReconciler) reconcileObjects(ctx context.Context, runtime jobruntimes.Runtime, trainJob *kubeflowv2.TrainJob) (objsOpState, error) {
	log := ctrl.LoggerFrom(ctx)

	objs, err := runtime.NewObjects(ctx, trainJob)
	if err != nil {
		return buildFailed, err
	}
	for _, obj := range objs {
		var gvk schema.GroupVersionKind
		if gvk, err = apiutil.GVKForObject(obj.DeepCopyObject(), r.client.Scheme()); err != nil {
			return buildFailed, err
		}
		logKeysAndValues := []any{
			"groupVersionKind", gvk.String(),
			"namespace", obj.GetNamespace(),
			"name", obj.GetName(),
		}
		// TODO (tenzen-y): Ideally, we should use the SSA instead of checking existence.
		// Non-empty resourceVersion indicates UPDATE operation.
		var creationErr error
		var created bool
		if obj.GetResourceVersion() == "" {
			creationErr = r.client.Create(ctx, obj)
			created = creationErr == nil
		}
		switch {
		case created:
			log.V(5).Info("Succeeded to create object", logKeysAndValues...)
			continue
		case client.IgnoreAlreadyExists(creationErr) != nil:
			return creationFailed, creationErr
		default:
			// This indicates CREATE operation has not been performed or the object has already existed in the cluster.
			if err = r.client.Update(ctx, obj); err != nil {
				return updateFailed, err
			}
			log.V(5).Info("Succeeded to update object", logKeysAndValues...)
		}
	}
	return creationSucceeded, nil
}

func setCreatedCondition(trainJob *kubeflowv2.TrainJob, opState objsOpState) {
	var newCond metav1.Condition
	switch opState {
	case creationSucceeded:
		newCond = metav1.Condition{
			Type:    kubeflowv2.TrainJobCreated,
			Status:  metav1.ConditionTrue,
			Message: constants.TrainJobJobsCreationSucceededMessage,
			Reason:  kubeflowv2.TrainJobJobsCreationSucceededReason,
		}
	case buildFailed:
		newCond = metav1.Condition{
			Type:    kubeflowv2.TrainJobCreated,
			Status:  metav1.ConditionFalse,
			Message: constants.TrainJobJobsBuildFailedMessage,
			Reason:  kubeflowv2.TrainJobJobsBuildFailedReason,
		}
	// TODO (tenzen-y): Provide more granular message based on creation or update failure.
	case creationFailed, updateFailed:
		newCond = metav1.Condition{
			Type:    kubeflowv2.TrainJobCreated,
			Status:  metav1.ConditionFalse,
			Message: constants.TrainJobJobsCreationFailedMessage,
			Reason:  kubeflowv2.TrainJobJobsCreationFailedReason,
		}
	default:
		return
	}
	meta.SetStatusCondition(&trainJob.Status.Conditions, newCond)
}

func setSuspendedCondition(trainJob *kubeflowv2.TrainJob) {
	var newCond metav1.Condition
	switch {
	case ptr.Deref(trainJob.Spec.Suspend, false):
		newCond = metav1.Condition{
			Type:    kubeflowv2.TrainJobSuspended,
			Status:  metav1.ConditionTrue,
			Message: constants.TrainJobSuspendedMessage,
			Reason:  kubeflowv2.TrainJobSuspendedReason,
		}
	case meta.IsStatusConditionTrue(trainJob.Status.Conditions, kubeflowv2.TrainJobSuspended):
		newCond = metav1.Condition{
			Type:    kubeflowv2.TrainJobSuspended,
			Status:  metav1.ConditionFalse,
			Message: constants.TrainJobResumedMessage,
			Reason:  kubeflowv2.TrainJobResumedReason,
		}
	default:
		return
	}
	meta.SetStatusCondition(&trainJob.Status.Conditions, newCond)
}

func setTerminalCondition(ctx context.Context, runtime jobruntimes.Runtime, trainJob *kubeflowv2.TrainJob) error {
	terminalCond, err := runtime.TerminalCondition(ctx, trainJob)
	if err != nil {
		return err
	}
	if terminalCond != nil {
		meta.SetStatusCondition(&trainJob.Status.Conditions, *terminalCond)
	}
	return nil
}

func isTrainJobFinished(trainJob *kubeflowv2.TrainJob) bool {
	return meta.IsStatusConditionTrue(trainJob.Status.Conditions, kubeflowv2.TrainJobComplete) ||
		meta.IsStatusConditionTrue(trainJob.Status.Conditions, kubeflowv2.TrainJobFailed)
}

func runtimeRefToGroupKind(runtimeRef kubeflowv2.RuntimeRef) schema.GroupKind {
	return schema.GroupKind{
		Group: ptr.Deref(runtimeRef.APIGroup, ""),
		Kind:  ptr.Deref(runtimeRef.Kind, ""),
	}
}

func (r *TrainJobReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	b := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&kubeflowv2.TrainJob{})
	for _, runtime := range r.runtimes {
		for _, registrar := range runtime.EventHandlerRegistrars() {
			if registrar != nil {
				b = registrar(b, mgr.GetClient(), mgr.GetCache())
			}
		}
	}
	return b.Complete(r)
}
