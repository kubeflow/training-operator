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

package pytorch

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/common/util"
	"github.com/kubeflow/training-operator/pkg/webhooks/utils"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
)

var (
	specPath               = field.NewPath("spec")
	pytorchReplicaSpecPath = specPath.Child("pytorchReplicaSpecs")
)

type Webhook struct{}

func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&trainingoperator.PyTorchJob{}).
		WithValidator(&Webhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-kubeflow-org-v1-pytorchjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeflow.org,resources=pytorchjobs,verbs=create;update,versions=v1,name=validator.pytorchjob.training-operator.kubeflow.org,admissionReviewVersions=v1

var _ webhook.CustomValidator = &Webhook{}

func (w *Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	job := obj.(*trainingoperator.PyTorchJob)
	log := ctrl.LoggerFrom(ctx).WithName("pytorchjob-webhook")
	log.V(5).Info("Validating create", "pytorchJob", klog.KObj(job))
	warnings, errs := validatePyTorchJob(nil, job)
	return warnings, errs.ToAggregate()
}

func (w *Webhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldJob := newObj.(*trainingoperator.PyTorchJob)
	newJob := newObj.(*trainingoperator.PyTorchJob)
	log := ctrl.LoggerFrom(ctx).WithName("pytorchjob-webhook")
	log.V(5).Info("Validating update", "pytorchJob", klog.KObj(newJob))
	warnings, errs := validatePyTorchJob(oldJob, newJob)
	return warnings, errs.ToAggregate()
}

func (w *Webhook) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validatePyTorchJob(oldJob, newJob *trainingoperator.PyTorchJob) (admission.Warnings, field.ErrorList) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	if errors := apimachineryvalidation.NameIsDNS1035Label(newJob.ObjectMeta.Name, false); len(errors) != 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("name"), newJob.Name, fmt.Sprintf("should match: %v", strings.Join(errors, ","))))
	}
	if oldJob != nil {
		allErrs = append(allErrs, util.ValidateRunPolicyUpdate(&oldJob.Spec.RunPolicy, &newJob.Spec.RunPolicy)...)
	}
	allErrs = append(allErrs, util.ValidateRunPolicy(&newJob.Spec.RunPolicy)...)
	ws, err := validateSpec(newJob.Spec)
	warnings = append(warnings, ws...)
	allErrs = append(allErrs, err...)
	return warnings, allErrs
}

func validateSpec(spec trainingoperator.PyTorchJobSpec) (admission.Warnings, field.ErrorList) {
	var allErrs field.ErrorList
	var warnings admission.Warnings
	if spec.ElasticPolicy != nil {
		_, ok := spec.PyTorchReplicaSpecs[trainingoperator.PyTorchJobReplicaTypeWorker]
		workerPath := pytorchReplicaSpecPath.Key(string(trainingoperator.PyTorchJobReplicaTypeWorker))
		if !ok {
			allErrs = append(allErrs, field.Required(workerPath, "must be configured if elastic policy is used"))
		}
		if spec.ElasticPolicy.NProcPerNode != nil {
			elasticNProcPerNodePath := specPath.Child("elasticPolicy").Child("nProcPerNode")
			nprocPerNodePath := specPath.Child("nprocPerNode")
			warnings = append(warnings, fmt.Sprintf("%s is deprecated, use %s instead", elasticNProcPerNodePath.String(), nprocPerNodePath.String()))
			if spec.NprocPerNode != nil {
				allErrs = append(allErrs, field.Forbidden(elasticNProcPerNodePath, fmt.Sprintf("must not be used with %s", nprocPerNodePath)))
			}
		}
	}
	allErrs = append(allErrs, validatePyTorchReplicaSpecs(spec.PyTorchReplicaSpecs)...)
	return warnings, allErrs
}

func validatePyTorchReplicaSpecs(rSpecs map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec) field.ErrorList {
	// Make sure the replica type is valid.
	validReplicaTypes := []trainingoperator.ReplicaType{
		trainingoperator.PyTorchJobReplicaTypeMaster,
		trainingoperator.PyTorchJobReplicaTypeWorker,
	}

	allErrs := utils.ValidateReplicaSpecs(rSpecs,
		trainingoperator.PyTorchJobDefaultContainerName,
		validReplicaTypes,
		pytorchReplicaSpecPath)

	for rType, rSpec := range rSpecs {
		if rType == trainingoperator.PyTorchJobReplicaTypeMaster {
			if rSpec.Replicas == nil || int(*rSpec.Replicas) != 1 {
				allErrs = append(allErrs, field.Forbidden(pytorchReplicaSpecPath.Key(string(rType)).Child("replicas"), "must be 1"))
			}
		} else if rSpec.Replicas != nil && int(*rSpec.Replicas) < 1 {
			allErrs = append(allErrs, field.Forbidden(rolePath.Child("replicas"), "must be at least 1"))
		}
	}
	return allErrs
}
