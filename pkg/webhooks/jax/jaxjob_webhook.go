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

package jax

import (
	"context"
	"fmt"
	"strings"

	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/webhooks/utils"
)

var (
	specPath           = field.NewPath("spec")
	jaxReplicaSpecPath = specPath.Child("jaxReplicaSpecs")
)

type Webhook struct{}

func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&trainingoperator.JAXJob{}).
		WithValidator(&Webhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-kubeflow-org-v1-jaxjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeflow.org,resources=jaxjobs,verbs=create;update,versions=v1,name=validator.jaxjob.training-operator.kubeflow.org,admissionReviewVersions=v1

var _ webhook.CustomValidator = &Webhook{}

func (w *Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	job := obj.(*trainingoperator.JAXJob)
	log := ctrl.LoggerFrom(ctx).WithName("jaxjob-webhook")
	log.V(5).Info("Validating create", "jaxJob", klog.KObj(job))
	return nil, validateJAXJob(job).ToAggregate()
}

func (w *Webhook) ValidateUpdate(ctx context.Context, _ runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	job := newObj.(*trainingoperator.JAXJob)
	log := ctrl.LoggerFrom(ctx).WithName("jaxjob-webhook")
	log.V(5).Info("Validating update", "jaxJob", klog.KObj(job))
	return nil, validateJAXJob(job).ToAggregate()
}

func (w *Webhook) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateJAXJob(job *trainingoperator.JAXJob) field.ErrorList {
	var allErrs field.ErrorList
	if errors := apimachineryvalidation.NameIsDNS1035Label(job.ObjectMeta.Name, false); len(errors) != 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("name"), job.Name, fmt.Sprintf("should match: %v", strings.Join(errors, ","))))
	}

	allErrs = append(allErrs, validateSpec(job.Spec)...)
	return allErrs
}

func validateSpec(spec trainingoperator.JAXJobSpec) field.ErrorList {
	return validateJAXReplicaSpecs(spec.JAXReplicaSpecs)
}

func validateJAXReplicaSpecs(rSpecs map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec) field.ErrorList {
	// Make sure the replica type is valid.
	validReplicaTypes := []trainingoperator.ReplicaType{
		trainingoperator.JAXJobReplicaTypeWorker,
	}

	return utils.ValidateReplicaSpecs(rSpecs,
		trainingoperator.JAXJobDefaultContainerName,
		validReplicaTypes,
		jaxReplicaSpecPath)
}
