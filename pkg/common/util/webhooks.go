package util

import (
	v1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"

	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var supportedJobControllers = sets.New(
	v1.MultiKueueController,
	v1.KubeflowJobsController)

func ValidateRunPolicyCreate(runPolicy *v1.RunPolicy) field.ErrorList {
	errs := field.ErrorList{}
	if runPolicy.ManagedBy != nil {
		manager := *runPolicy.ManagedBy
		if !supportedJobControllers.Has(manager) {
			fieldPath := field.NewPath("spec", "runPolicy", "managedBy")
			errs = append(errs, field.NotSupported(fieldPath, manager, supportedJobControllers.UnsortedList()))
		}
	}
	return errs
}

func ValidateRunPolicyUpdate(oldRunPolicy, newRunPolicy *v1.RunPolicy) field.ErrorList {
	oldManager := oldRunPolicy.ManagedBy
	newManager := newRunPolicy.ManagedBy
	fieldPath := field.NewPath("spec", "runPolicy", "managedBy")
	return apivalidation.ValidateImmutableField(newManager, oldManager, fieldPath)
}
