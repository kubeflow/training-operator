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

package webhooksv2

import (
	"context"

	apiruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/trainer/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
)

type TrainJobWebhook struct {
	runtimes map[string]runtime.Runtime
}

func setupWebhookForTrainJob(mgr ctrl.Manager, run map[string]runtime.Runtime) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kubeflowv2.TrainJob{}).
		WithValidator(&TrainJobWebhook{runtimes: run}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-kubeflow-org-v2alpha1-trainjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeflow.org,resources=trainjobs,verbs=create;update,versions=v2alpha1,name=validator.trainjob.kubeflow.org,admissionReviewVersions=v1

var _ webhook.CustomValidator = (*TrainJobWebhook)(nil)

func (w *TrainJobWebhook) ValidateCreate(context.Context, apiruntime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (w *TrainJobWebhook) ValidateUpdate(context.Context, apiruntime.Object, apiruntime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (w *TrainJobWebhook) ValidateDelete(context.Context, apiruntime.Object) (admission.Warnings, error) {
	return nil, nil
}
