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

package paddlepaddle

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
	specPath              = field.NewPath("spec")
	paddleReplicaSpecPath = specPath.Child("paddleReplicaSpecs")
)

type Webhook struct{}

func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&trainingoperator.PaddleJob{}).
		WithValidator(&Webhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-kubeflow-org-v1-paddlejob,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeflow.org,resources=paddlejobs,verbs=create;update,versions=v1,name=validator.paddlejob.training-operator.kubeflow.org,admissionReviewVersions=v1

var _ webhook.CustomValidator = &Webhook{}

func (w Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	job := obj.(*trainingoperator.PaddleJob)
	log := ctrl.LoggerFrom(ctx).WithName("paddlejob-webhook")
	log.V(5).Info("Validating create", "paddleJob", klog.KObj(job))
	return nil, validatePaddleJob(nil, job).ToAggregate()
}

func (w Webhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldJob := oldObj.(*trainingoperator.PaddleJob)
	newJob := newObj.(*trainingoperator.PaddleJob)
	log := ctrl.LoggerFrom(ctx).WithName("paddlejob-webhook")
	log.V(5).Info("Validating update", "paddleJob", klog.KObj(newJob))
	return nil, validatePaddleJob(oldJob, newJob).ToAggregate()
}

func (w Webhook) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validatePaddleJob(oldJob, newJob *trainingoperator.PaddleJob) field.ErrorList {
	var allErrs field.ErrorList
	if errors := apimachineryvalidation.NameIsDNS1035Label(newJob.Name, false); len(errors) != 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("name"), newJob.Name, fmt.Sprintf("should match: %v", strings.Join(errors, ","))))
	}
	if oldJob != nil {
		allErrs = append(allErrs, util.ValidateRunPolicyUpdate(&oldJob.Spec.RunPolicy, &newJob.Spec.RunPolicy)...)
	}
	allErrs = append(allErrs, util.ValidateRunPolicy(&newJob.Spec.RunPolicy)...)
	allErrs = append(allErrs, validateSpec(newJob.Spec.PaddleReplicaSpecs)...)
	return allErrs
}

func validateSpec(rSpecs map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec) field.ErrorList {
	// Make sure the replica type is valid.
	validReplicaTypes := []trainingoperator.ReplicaType{
		trainingoperator.PaddleJobReplicaTypeMaster,
		trainingoperator.PaddleJobReplicaTypeWorker,
	}

	return utils.ValidateReplicaSpecs(rSpecs,
		trainingoperator.PaddleJobDefaultContainerName,
		validReplicaTypes,
		paddleReplicaSpecPath)
}
