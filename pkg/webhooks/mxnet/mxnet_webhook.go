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

package mxnet

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

type Webhook struct{}

func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&trainingoperator.MXJob{}).
		WithValidator(&Webhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-kubeflow-org-v1-mxjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeflow.org,resources=mxjobs,verbs=create;update;delete,versions=v1,name=validator.mxjob.training-operator.kubeflow.org,admissionReviewVersions=v1

var _ webhook.CustomValidator = &Webhook{}

func (w *Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	job := obj.(*trainingoperator.MXJob)
	log := ctrl.LoggerFrom(ctx).WithName("mxjob-webhook")
	log.V(5).Info("Validating create", "mxJob", klog.KObj(job))
	return deprecatedWarning(), nil
}

func (w *Webhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	job := newObj.(*trainingoperator.MXJob)
	log := ctrl.LoggerFrom(ctx).WithName("mxjob-webhook")
	log.V(5).Info("Validating update", "mxJob", klog.KObj(job))
	return deprecatedWarning(), nil
}

func (w *Webhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	job := obj.(*trainingoperator.MXJob)
	log := ctrl.LoggerFrom(ctx).WithName("mxjob-webhook")
	log.V(5).Info("Validating delete", "mxJob", klog.KObj(job))
	return deprecatedWarning(), nil
}

func deprecatedWarning() admission.Warnings {
	return admission.Warnings{
		fmt.Sprintf("MXJob is deprecated, and the MXJob will be removed in the release v1.9.0.\n" +
			"Please see https://github.com/kubeflow/training-operator/issues/1996 for more details"),
	}
}
