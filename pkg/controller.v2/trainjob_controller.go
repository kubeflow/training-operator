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
	"fmt"
	runtimeUtils "github.com/kubeflow/training-operator/pkg/util.v2/runtime"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	jobruntimes "github.com/kubeflow/training-operator/pkg/runtime.v2"
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

//+kubebuilder:rbac:groups=kubeflow.org,resources=trainjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=trainjobs/status,verbs=get;update;patch

func (r *TrainJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var trainJob kubeflowv2.TrainJob
	if err := r.client.Get(ctx, req.NamespacedName, &trainJob); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log := ctrl.LoggerFrom(ctx).WithValues("trainJob", klog.KObj(&trainJob))
	ctx = ctrl.LoggerInto(ctx, log)
	log.V(2).Info("Reconciling TrainJob")
	if err := r.createOrUpdateObjs(ctx, &trainJob); err != nil {
		return ctrl.Result{}, err
	}
	// TODO (tenzen-y): Do update the status.
	return ctrl.Result{}, nil
}

func (r *TrainJobReconciler) createOrUpdateObjs(ctx context.Context, trainJob *kubeflowv2.TrainJob) error {
	log := ctrl.LoggerFrom(ctx)

	runtimeRefGK := runtimeUtils.RuntimeRefToGroupKind(trainJob.Spec.RuntimeRef).String()
	runtime, ok := r.runtimes[runtimeRefGK]
	if !ok {
		return fmt.Errorf("%w: %s", runtimeUtils.ErrorUnsupportedRuntime, runtimeRefGK)
	}
	objs, err := runtime.NewObjects(ctx, trainJob)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		var gvk schema.GroupVersionKind
		if gvk, err = apiutil.GVKForObject(obj.DeepCopyObject(), r.client.Scheme()); err != nil {
			return err
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
			log.V(5).Info("Succeeded to create object", logKeysAndValues)
			continue
		case client.IgnoreAlreadyExists(creationErr) != nil:
			return creationErr
		default:
			// This indicates CREATE operation has not been performed or the object has already existed in the cluster.
			if err = r.client.Update(ctx, obj); err != nil {
				return err
			}
			log.V(5).Info("Succeeded to update object", logKeysAndValues)
		}
	}
	return nil
}

func (r *TrainJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&kubeflowv2.TrainJob{})
	for _, runtime := range r.runtimes {
		for _, registrar := range runtime.EventHandlerRegistrars() {
			if registrar != nil {
				b = registrar(b, mgr.GetClient())
			}
		}
	}
	return b.Complete(r)
}
