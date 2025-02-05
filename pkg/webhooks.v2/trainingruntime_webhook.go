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
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/trainer/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
)

type TrainingRuntimeWebhook struct {
	runtimes map[string]runtime.Runtime
}

func setupWebhookForTrainingRuntime(mgr ctrl.Manager, run map[string]runtime.Runtime) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kubeflowv2.TrainingRuntime{}).
		WithValidator(&TrainingRuntimeWebhook{runtimes: run}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-kubeflow-org-v2alpha1-trainingruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeflow.org,resources=trainingruntimes,verbs=create;update,versions=v2alpha1,name=validator.trainingruntime.kubeflow.org,admissionReviewVersions=v1

var _ webhook.CustomValidator = (*TrainingRuntimeWebhook)(nil)

func (w *TrainingRuntimeWebhook) ValidateCreate(ctx context.Context, obj apiruntime.Object) (admission.Warnings, error) {
	trainingRuntime := obj.(*kubeflowv2.TrainingRuntime)
	log := ctrl.LoggerFrom(ctx).WithName("trainingruntime-webhook")
	log.V(5).Info("Validating create", "trainingRuntime", klog.KObj(trainingRuntime))
	return nil, validateReplicatedJobs(trainingRuntime.Spec.Template.Spec.ReplicatedJobs).ToAggregate()
}

func validateReplicatedJobs(rJobs []jobsetv1alpha2.ReplicatedJob) field.ErrorList {
	rJobsPath := field.NewPath("spec").
		Child("template").
		Child("spec").
		Child("replicatedJobs")
	var allErrs field.ErrorList
	for idx, rJob := range rJobs {
		if rJob.Replicas != 1 {
			allErrs = append(allErrs, field.Invalid(rJobsPath.Index(idx).Child("replicas"), rJob.Replicas, "always must be 1"))
		}
	}
	return allErrs
}

func (w *TrainingRuntimeWebhook) ValidateUpdate(context.Context, apiruntime.Object, apiruntime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (w *TrainingRuntimeWebhook) ValidateDelete(context.Context, apiruntime.Object) (admission.Warnings, error) {
	return nil, nil
}
