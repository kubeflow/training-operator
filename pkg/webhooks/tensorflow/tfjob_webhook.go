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

package tensorflow

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/common/util"
	"github.com/kubeflow/training-operator/pkg/webhooks/utils"
)

var (
	specPath          = field.NewPath("spec")
	tfReplicaSpecPath = specPath.Child("tfReplicaSpecs")
)

type Webhook struct{}

func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&trainingoperator.TFJob{}).
		WithValidator(&Webhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-kubeflow-org-v1-tfjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeflow.org,resources=tfjobs,verbs=create;update,versions=v1,name=validator.tfjob.training-operator.kubeflow.org,admissionReviewVersions=v1

var _ webhook.CustomValidator = &Webhook{}

func (w *Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	job := obj.(*trainingoperator.TFJob)
	log := ctrl.LoggerFrom(ctx).WithName("tfjob-webhook")
	log.V(5).Info("Validating create", "TFJob", klog.KObj(job))
	return nil, validateTFJob(nil, job).ToAggregate()
}

func (w *Webhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldJob := oldObj.(*trainingoperator.TFJob)
	newJob := newObj.(*trainingoperator.TFJob)
	log := ctrl.LoggerFrom(ctx).WithName("tfjob-webhook")
	log.V(5).Info("Validating update", "NewTFJob", klog.KObj(newJob))
	return nil, validateTFJob(oldJob, newJob).ToAggregate()
}

func (w *Webhook) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateTFJob(oldJob, newJob *trainingoperator.TFJob) field.ErrorList {
	var allErrs field.ErrorList
	err := utils.ValidateJobName(newJob.Name)
	if err != nil {
		allErrs = append(allErrs, err)
	}
	if oldJob != nil {
		allErrs = append(allErrs, util.ValidateRunPolicyUpdate(&oldJob.Spec.RunPolicy, &newJob.Spec.RunPolicy)...)
	}
	allErrs = append(allErrs, util.ValidateRunPolicy(&newJob.Spec.RunPolicy)...)
	allErrs = append(allErrs, validateSpec(newJob.Spec)...)
	return allErrs
}

func validateSpec(spec trainingoperator.TFJobSpec) field.ErrorList {
	return validateTFReplicaSpecs(spec.TFReplicaSpecs)
}

func validateTFReplicaSpecs(rSpecs map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec) field.ErrorList {
	allErrs := utils.ValidateReplicaSpecs(rSpecs,
		trainingoperator.TFJobDefaultContainerName,
		nil,
		tfReplicaSpecPath)

	chiefOrMaster := 0
	for rType := range rSpecs {
		if trainingoperator.IsChiefOrMaster(rType) {
			chiefOrMaster++
		}
	}
	if chiefOrMaster > 1 {
		allErrs = append(allErrs, field.Forbidden(tfReplicaSpecPath, "must not have more than 1 Chief or Master role"))
	}
	return allErrs
}
