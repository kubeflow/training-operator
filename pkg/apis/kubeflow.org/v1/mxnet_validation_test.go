// Copyright 2021 The Kubeflow Authors
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
// limitations under the License

package v1

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	v1 "k8s.io/api/core/v1"
)

func TestValidateV1MXJobSpec(t *testing.T) {
	testCases := []MXJobSpec{
		{
			MXReplicaSpecs: nil,
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "",
								},
							},
						},
					},
				},
			},
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Name:  "",
									Image: "mxjob/mxnet:gpu",
								},
							},
						},
					},
				},
			},
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				MXJobReplicaTypeScheduler: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
	}
	for _, c := range testCases {
		err := ValidateV1MXJobSpec(&c)
		if err.Error() != "MXJobSpec is not valid" {
			t.Error("Failed validate the alpha2.MXJobSpec")
		}
	}
}
