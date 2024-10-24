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

package mpi

import (
	"context"
	"strconv"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework"
)

type MPI struct {
	client client.Client
}

var _ framework.EnforceMLPolicyPlugin = (*MPI)(nil)
var _ framework.CustomValidationPlugin = (*MPI)(nil)

const Name = "MPI"

func New(_ context.Context, client client.Client, _ client.FieldIndexer) (framework.Plugin, error) {
	return &MPI{
		client: client,
	}, nil
}

func (m *MPI) Name() string {
	return Name
}

func (m *MPI) EnforceMLPolicy(info *runtime.Info) error {
	if info == nil || info.MLPolicy == nil || info.MLPolicy.MPI == nil {
		return nil
	}
	// TODO: Need to implement main logic.
	return nil
}

func (m *MPI) Validate(oldJobObj, newJobObj *kubeflowv2.TrainJob, runtimeInfo *runtime.Info) (admission.Warnings, field.ErrorList) {
	var allErrs field.ErrorList
	specPath := field.NewPath("spec")
	if newJobObj.Spec.Trainer != nil {
		numProcPerNodePath := specPath.Child("trainer").Child("numProcPerNode")
		if runtimeInfo.MLPolicy.MPI != nil {
			if _, err := strconv.Atoi(*newJobObj.Spec.Trainer.NumProcPerNode); err != nil {
				allErrs = append(allErrs, field.Invalid(numProcPerNodePath, newJobObj.Spec.Trainer.NumProcPerNode, "should have an int value"))
			}
		}
	}
	return nil, allErrs
}
