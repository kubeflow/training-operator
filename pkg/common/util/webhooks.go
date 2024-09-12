package util

import (
	v1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var supportedJobControllers = sets.New(
	v1.MultiKueueController,
	v1.KubeflowJobsController)

func ValidateManagedBy(runPolicy *v1.RunPolicy, allErrs field.ErrorList) field.ErrorList {
	if runPolicy.ManagedBy != nil {
		manager := *runPolicy.ManagedBy
		if !supportedJobControllers.Has(manager) {
			fieldPath := field.NewPath("spec", "managedBy")
			allErrs = append(allErrs, field.NotSupported(fieldPath, manager, supportedJobControllers.UnsortedList()))
		}
	}
	return allErrs
}
