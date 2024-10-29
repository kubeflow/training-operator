package utils

import (
	"fmt"
	"strings"

	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateJobName(name string) *field.Error {
	if errors := apimachineryvalidation.NameIsDNS1035Label(name, false); len(errors) != 0 {
		return field.Invalid(field.NewPath("metadata").Child("name"), name, fmt.Sprintf("should match: %v", strings.Join(errors, ",")))
	}
	return nil
}
