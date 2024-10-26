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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	"github.com/kubeflow/training-operator/pkg/constants"
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

// TODO (andreyvelich): Add support for PyTorch elastic when JobSet supports Elastic Jobs.
func (t *Torch) EnforceMLPolicy(info *runtime.Info, trainJob *kubeflowv2.TrainJob) error {
	if info == nil || info.MLPolicy == nil || info.MLPolicy.Torch == nil {
		return nil
	}

	// TrainJob contains the actual information for the Trainer.
	numNodes := info.MLPolicy.NumNodes
	if trainJob.Spec.Trainer != nil && trainJob.Spec.Trainer.NumNodes != nil {
		numNodes = trainJob.Spec.Trainer.NumNodes
	}
	info.Trainer.NumNodes = numNodes

	numProcPerNode := info.MLPolicy.Torch.NumProcPerNode
	if trainJob.Spec.Trainer != nil && trainJob.Spec.Trainer.NumProcPerNode != nil {
		numProcPerNode = trainJob.Spec.Trainer.NumProcPerNode
	}

	// Update envs for Info object.
	// Add PyTorch distributed "PET_" values for torchrun
	// TODO (andreyvelich): Add validation to check that TrainJob doesn't have "PET_" envs.
	info.Trainer.Env = []corev1.EnvVar{
		{
			Name:  constants.TorchEnvNumNodes,
			Value: fmt.Sprintf("%d", *numNodes),
		},
		{
			Name:  constants.TorchEnvNumProcPerNode,
			Value: *numProcPerNode,
		},
		{
			Name: constants.TorchEnvNodeRank,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: constants.JobCompletionIndexFieldPath,
				},
			},
		},
		{
			Name:  constants.TorchEnvMasterAddr,
			Value: fmt.Sprintf("%v-%v-0-0.%v", trainJob.Name, constants.JobTrainerNode, constants.ContainerTrainer),
		},
		{
			Name:  constants.TorchEnvMasterPort,
			Value: fmt.Sprintf("%d", constants.ContainerTrainerPort),
		},
	}

	// Map for all Info envs.
	envNames := make(map[string]bool, len(info.Trainer.Env))
	for _, env := range info.Trainer.Env {
		envNames[env.Name] = true
	}
	// Info envs take precedence over TrainJob envs.
	if trainJob.Spec.Trainer != nil {
		for _, env := range trainJob.Spec.Trainer.Env {
			if _, ok := envNames[env.Name]; !ok {
				info.Trainer.Env = append(info.Trainer.Env, corev1.EnvVar{Name: env.Name, Value: env.Value})
			}
		}
	}

	// Add container port for the headless service.
	info.Trainer.ContainerPort = &corev1.ContainerPort{
		ContainerPort: constants.ContainerTrainerPort,
	}

	// Update total Pod requests for the PodGroupPolicy plugin.
	for rName := range info.TotalRequests {
		// For other Jobs like the Initializer, replica is always equal to 1.
		// TODO (andreyvelich): Add support for total requests from the TrainJob's ResourcesPerNode.
		if rName == constants.JobTrainerNode {
			info.TotalRequests[rName] = runtime.TotalResourceRequest{
				Replicas:    *numNodes,
				PodRequests: info.TotalRequests[rName].PodRequests,
			}
		}
	}

	return nil
}

// TODO: Need to implement validateions for TorchJob.
func (t *Torch) Validate(oldObj, newObj *kubeflowv2.TrainJob) (admission.Warnings, field.ErrorList) {
	return nil, nil
}
