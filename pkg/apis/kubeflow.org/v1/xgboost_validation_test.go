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
// limitations under the License.

package v1

import (
	"k8s.io/utils/pointer"
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateXGBoostJobSpec(t *testing.T) {
	testCases := []XGBoostJobSpec{
		{
			XGBReplicaSpecs: nil,
		},
		{
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{},
						},
					},
				},
			},
		},
		{
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
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
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name:  "",
									Image: "gcr.io/kubeflow-ci/xgboost-dist-mnist_test:1.0",
								},
							},
						},
					},
				},
			},
		},
		{
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				XGBoostJobReplicaTypeMaster: &commonv1.ReplicaSpec{
					Replicas: pointer.Int32(2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name:  "xgboost",
									Image: "gcr.io/kubeflow-ci/xgboost-dist-mnist_test:1.0",
								},
							},
						},
					},
				},
			},
		},
		{
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Replicas: pointer.Int32(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name:  "xgboost",
									Image: "gcr.io/kubeflow-ci/xgboost-dist-mnist_test:1.0",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, c := range testCases {
		err := ValidateXGBoostJobSpec(&c)
		if err == nil {
			t.Error("Failed validate the corev1.XGBoostJobSpec")
		}
	}
}
