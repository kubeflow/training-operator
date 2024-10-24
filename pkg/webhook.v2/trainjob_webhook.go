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

package webhookv2

import (
	"context"
	"fmt"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	jobRuntime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	runtimeUtils "github.com/kubeflow/training-operator/pkg/util.v2/runtime"
)

type TrainJobWebhook struct {
	runtimes map[string]jobRuntime.Runtime
}

func setupWebhookForTrainJob(mgr ctrl.Manager, run map[string]jobRuntime.Runtime) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kubeflowv2.TrainJob{}).
		WithValidator(&TrainJobWebhook{runtimes: run}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-kubeflow-org-v2alpha1-trainjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeflow.org,resources=trainjobs,verbs=create;update,versions=v2alpha1,name=validator.trainjob.kubeflow.org,admissionReviewVersions=v1

var _ webhook.CustomValidator = (*TrainJobWebhook)(nil)

func (w *TrainJobWebhook) ValidateCreate(ctx context.Context, obj apiruntime.Object) (admission.Warnings, error) {
	trainJob := obj.(*kubeflowv2.TrainJob)
	log := ctrl.LoggerFrom(ctx).WithName("trainJob-webhook")
	log.V(5).Info("Validating create", "TrainJob", klog.KObj(trainJob))
	runtimeRefGK := runtimeUtils.RuntimeRefToGroupKind(trainJob.Spec.RuntimeRef).String()
	runtime, ok := w.runtimes[runtimeRefGK]
	if !ok {
		return nil, fmt.Errorf("%w: %s", runtimeUtils.ErrorUnsupportedRuntime, runtimeRefGK)
	}
	warnings, errorList := runtime.ValidateObjects(ctx, nil, trainJob)
	return warnings, errorList.ToAggregate()
}

func (w *TrainJobWebhook) ValidateUpdate(ctx context.Context, oldObj apiruntime.Object, newObj apiruntime.Object) (admission.Warnings, error) {
	oldTrainJob := oldObj.(*kubeflowv2.TrainJob)
	newTrainJob := newObj.(*kubeflowv2.TrainJob)
	log := ctrl.LoggerFrom(ctx).WithName("trainJob-webhook")
	log.V(5).Info("Validating update", "TrainJob", klog.KObj(newTrainJob))
	runtimeRefGK := runtimeUtils.RuntimeRefToGroupKind(newTrainJob.Spec.RuntimeRef).String()
	runtime, ok := w.runtimes[runtimeRefGK]
	if !ok {
		return nil, fmt.Errorf("%w: %s", runtimeUtils.ErrorUnsupportedRuntime, runtimeRefGK)
	}
	warnings, errorList := runtime.ValidateObjects(ctx, oldTrainJob, newTrainJob)
	return warnings, errorList.ToAggregate()
}

func (w *TrainJobWebhook) ValidateDelete(context.Context, apiruntime.Object) (admission.Warnings, error) {
	return nil, nil
}
