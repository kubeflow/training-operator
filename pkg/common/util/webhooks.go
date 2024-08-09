package util

import (
	v1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateManagedBy(runPolicy *v1.RunPolicy, allErrs field.ErrorList) field.ErrorList {
	if runPolicy.ManagedBy != nil {
		manager := *runPolicy.ManagedBy
		fieldPath := field.NewPath("spec", "managedBy")
		allErrs = append(allErrs, validation.IsDomainPrefixedPath(fieldPath, manager)...)
		if len(manager) > v1.MaxManagedByLength {
			allErrs = append(allErrs, field.TooLongMaxLength(fieldPath, manager, v1.MaxManagedByLength))
		}
	}
	return allErrs
}
