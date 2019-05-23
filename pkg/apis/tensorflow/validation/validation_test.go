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

package validation

import (
	"testing"

	"github.com/golang/protobuf/proto"
	commonv1 "github.com/kubeflow/tf-operator/pkg/apis/common/v1"
	commonv1beta2 "github.com/kubeflow/tf-operator/pkg/apis/common/v1beta2"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	tfv1beta2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1beta2"

	"k8s.io/api/core/v1"
)

func TestValidateBetaTwoTFJobSpec(t *testing.T) {
	testCases := []tfv1beta2.TFJobSpec{
		{
			TFReplicaSpecs: nil,
		},
		{
			TFReplicaSpecs: map[tfv1beta2.TFReplicaType]*commonv1beta2.ReplicaSpec{
				tfv1beta2.TFReplicaTypeWorker: &commonv1beta2.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[tfv1beta2.TFReplicaType]*commonv1beta2.ReplicaSpec{
				tfv1beta2.TFReplicaTypeWorker: &commonv1beta2.ReplicaSpec{
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
			TFReplicaSpecs: map[tfv1beta2.TFReplicaType]*commonv1beta2.ReplicaSpec{
				tfv1beta2.TFReplicaTypeWorker: &commonv1beta2.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
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
			TFReplicaSpecs: map[tfv1beta2.TFReplicaType]*commonv1beta2.ReplicaSpec{
				tfv1beta2.TFReplicaTypeChief: &commonv1beta2.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
				tfv1beta2.TFReplicaTypeMaster: &commonv1beta2.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[tfv1beta2.TFReplicaType]*commonv1beta2.ReplicaSpec{
				tfv1beta2.TFReplicaTypeEval: &commonv1beta2.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Name:  "tensorflow",
									Image: "kubeflow/tf-dist-mnist-test:1.0",
								},
							},
						},
					},
					Replicas: proto.Int32(2),
				},
			},
		},
	}
	for _, c := range testCases {
		err := ValidateBetaTwoTFJobSpec(&c)
		if err == nil {
			t.Error("Expected error got nil")
		}
	}
}

func TestValidateV1TFJobSpec(t *testing.T) {
	testCases := []tfv1.TFJobSpec{
		{
			TFReplicaSpecs: nil,
		},
		{
			TFReplicaSpecs: map[tfv1.TFReplicaType]*commonv1.ReplicaSpec{
				tfv1.TFReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[tfv1.TFReplicaType]*commonv1.ReplicaSpec{
				tfv1.TFReplicaTypeWorker: &commonv1.ReplicaSpec{
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
			TFReplicaSpecs: map[tfv1.TFReplicaType]*commonv1.ReplicaSpec{
				tfv1.TFReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
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
			TFReplicaSpecs: map[tfv1.TFReplicaType]*commonv1.ReplicaSpec{
				tfv1.TFReplicaTypeChief: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
				tfv1.TFReplicaTypeMaster: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[tfv1.TFReplicaType]*commonv1.ReplicaSpec{
				tfv1.TFReplicaTypeEval: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Name:  "tensorflow",
									Image: "kubeflow/tf-dist-mnist-test:1.0",
								},
							},
						},
					},
					Replicas: proto.Int32(2),
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
