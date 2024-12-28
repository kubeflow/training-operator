package utils

import (
	"fmt"
	"slices"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateReplicaSpecs(rSpecs map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec,
	defaultContainerName string,
	validReplicaTypes []trainingoperator.ReplicaType,
	replicaSpecPath *field.Path) field.ErrorList {

	var allErrs field.ErrorList

	if rSpecs == nil {
		allErrs = append(allErrs, field.Required(replicaSpecPath, "must be required"))
	}

	for rType, rSpec := range rSpecs {
		rolePath := replicaSpecPath.Key(string(rType))
		containersPath := rolePath.Child("template").Child("spec").Child("containers")

		if len(validReplicaTypes) > 0 {
			if !slices.Contains(validReplicaTypes, rType) {
				allErrs = append(allErrs, field.NotSupported(rolePath, rType, validReplicaTypes))
			}
		}

		if rSpec == nil || len(rSpec.Template.Spec.Containers) == 0 {
			allErrs = append(allErrs, field.Required(containersPath, "must be specified"))
		}

		// Make sure the image is defined in the container
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
