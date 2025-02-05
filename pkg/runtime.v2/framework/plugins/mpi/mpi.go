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

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/trainer/v2alpha1"
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

func (m *MPI) EnforceMLPolicy(info *runtime.Info, trainJob *kubeflowv2.TrainJob) error {
	if info == nil || info.RuntimePolicy.MLPolicy == nil || info.RuntimePolicy.MLPolicy.MPI == nil {
		return nil
	}
	// TODO: Need to implement main logic.
	return nil
}

// TODO: Need to implement validations for MPIJob.
func (m *MPI) Validate(oldObj, newObj *kubeflowv2.TrainJob) (admission.Warnings, field.ErrorList) {
	return nil, nil
}
