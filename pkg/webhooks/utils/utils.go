package utils

import (
	"fmt"
	"slices"
	"strings"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateJobName(name string) *field.Error {
	if errors := apimachineryvalidation.NameIsDNS1035Label(name, false); len(errors) != 0 {
		return field.Invalid(field.NewPath("metadata").Child("name"), name, fmt.Sprintf("should match: %v", strings.Join(errors, ",")))
	}
	return nil
}

func ValidateReplicaSpecs(rSpecs map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec,
	defaultContainerName string,
	validRoleTypes []trainingoperator.ReplicaType,
	replicaSpecPath *field.Path) field.ErrorList {

	var allErrs field.ErrorList

	if rSpecs == nil {
		allErrs = append(allErrs, field.Required(replicaSpecPath, "must be required"))
	}

	for rType, rSpec := range rSpecs {
		rolePath := replicaSpecPath.Key(string(rType))
		containersPath := rolePath.Child("template").Child("spec").Child("containers")

		if len(validRoleTypes) > 0 {
			if !slices.Contains(validRoleTypes, rType) {
				allErrs = append(allErrs, field.NotSupported(rolePath, rType, validRoleTypes))
			}
		}

		if rSpec == nil || len(rSpec.Template.Spec.Containers) == 0 {
			allErrs = append(allErrs, field.Required(containersPath, "must be specified"))
		}

		defaultContainerPresent := false
		for idx, container := range rSpec.Template.Spec.Containers {
			if container.Image == "" {
				allErrs = append(allErrs, field.Required(containersPath.Index(idx).Child("image"), "must be required"))
			}
			if container.Name == defaultContainerName {
				defaultContainerPresent = true
			}
		}

		if !defaultContainerPresent {
			allErrs = append(allErrs, field.Required(containersPath,
				fmt.Sprintf("must have at least one container with name %s", defaultContainerName)))
		}
	}

	return allErrs
}
