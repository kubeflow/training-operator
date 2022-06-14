// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"k8s.io/utils/pointer"
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	v1 "k8s.io/api/core/v1"
)

func TestValidateV1PyTorchJobSpec(t *testing.T) {
	testCases := []PyTorchJobSpec{
		{
			PyTorchReplicaSpecs: nil,
		},
		{
			PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				PyTorchJobReplicaTypeWorker: {
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				PyTorchJobReplicaTypeWorker: {
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Image: "",
								},
							},
						},
					},
				},
			},
		},
		{
			PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				PyTorchJobReplicaTypeWorker: {
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "",
									Image: "gcr.io/kubeflow-ci/pytorch-dist-mnist_test:1.0",
								},
							},
						},
					},
				},
			},
		},
		{
			PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				PyTorchJobReplicaTypeMaster: {
					Replicas: pointer.Int32(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "pytorch",
									Image: "gcr.io/kubeflow-ci/pytorch-dist-mnist_test:1.0",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, c := range testCases {
		err := ValidateV1PyTorchJobSpec(&c)
		if err == nil {
			t.Error("Failed validate the v1.PyTorchJobSpec")
		}
	}
}
