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
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateV1TFJobSpec(t *testing.T) {
	testCases := []TFJobSpec{
		{
			TFReplicaSpecs: nil,
		},
		{
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				TFJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				TFJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Image: "",
								},
							},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				TFJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name:  "",
									Image: "kubeflow/tf-dist-mnist-test:1.0",
								},
							},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				TFJobReplicaTypeChief: &commonv1.ReplicaSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{},
						},
					},
				},
				TFJobReplicaTypeMaster: &commonv1.ReplicaSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{},
						},
					},
				},
			},
		},
	}
	for _, c := range testCases {
		err := ValidateV1TFJobSpec(&c)
		if err == nil {
			t.Error("Expected error got nil")
		}
	}
}

func TestIsChieforMaster(t *testing.T) {
	tc := []struct {
		Type     commonv1.ReplicaType
		Expected bool
	}{
		{
			Type:     TFJobReplicaTypeChief,
			Expected: true,
		},
		{
			Type:     TFJobReplicaTypeMaster,
			Expected: true,
		},
		{
			Type:     TFJobReplicaTypeWorker,
			Expected: false,
		},
	}

	for _, c := range tc {
		actual := IsChieforMaster(c.Type)
		if actual != c.Expected {
			t.Errorf("Expected %v; Got %v", c.Expected, actual)
		}
	}
}
