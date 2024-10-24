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

package torch

import (
	"context"
	"k8s.io/utils/strings/slices"
	"strconv"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework"
)

type Torch struct{}

var _ framework.EnforceMLPolicyPlugin = (*Torch)(nil)
var _ framework.CustomValidationPlugin = (*Torch)(nil)

const Name = "Torch"

func New(context.Context, client.Client, client.FieldIndexer) (framework.Plugin, error) {
	return &Torch{}, nil
}

func (t *Torch) Name() string {
	return Name
}

func (t *Torch) EnforceMLPolicy(info *runtime.Info) error {
	if info == nil || info.MLPolicy == nil || info.MLPolicy.Torch == nil {
		return nil
	}
	// TODO: Need to implement main logic.
	return nil
}

func (t *Torch) Validate(oldObj, newObj *kubeflowv2.TrainJob, runtimeInfo *runtime.Info) (admission.Warnings, field.ErrorList) {
	var allErrs field.ErrorList
	specPath := field.NewPath("spec")

	if newObj.Spec.Trainer != nil {
		numProcPerNodePath := specPath.Child("trainer").Child("numProcPerNode")
		if runtimeInfo.MLPolicy.Torch != nil {
			allowedStringValList := []string{"auto", "cpu", "gpu"}
			if !slices.Contains(allowedStringValList, *newObj.Spec.Trainer.NumProcPerNode) {
				if _, err := strconv.Atoi(*newObj.Spec.Trainer.NumProcPerNode); err != nil {
					allErrs = append(allErrs, field.Invalid(numProcPerNodePath, newObj.Spec.Trainer.NumProcPerNode, "should have an int value or auto/cpu/gpu"))
				}
			}
		}
	}
	return nil, allErrs
}
