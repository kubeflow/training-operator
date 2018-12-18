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
	common "github.com/kubeflow/tf-operator/pkg/apis/common/v1beta1"
	tfv2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfv1beta1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1beta1"

	"k8s.io/api/core/v1"
)

func TestValidateAlphaTwoTFJobSpec(t *testing.T) {
	testCases := []tfv2.TFJobSpec{
		{
			TFReplicaSpecs: nil,
		},
		{
			TFReplicaSpecs: map[tfv2.TFReplicaType]*tfv2.TFReplicaSpec{
				tfv2.TFReplicaTypeWorker: &tfv2.TFReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[tfv2.TFReplicaType]*tfv2.TFReplicaSpec{
				tfv2.TFReplicaTypeWorker: &tfv2.TFReplicaSpec{
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
			TFReplicaSpecs: map[tfv2.TFReplicaType]*tfv2.TFReplicaSpec{
				tfv2.TFReplicaTypeWorker: &tfv2.TFReplicaSpec{
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
			TFReplicaSpecs: map[tfv2.TFReplicaType]*tfv2.TFReplicaSpec{
				tfv2.TFReplicaTypeChief: &tfv2.TFReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
				tfv2.TFReplicaTypeMaster: &tfv2.TFReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[tfv2.TFReplicaType]*tfv2.TFReplicaSpec{
				tfv2.TFReplicaTypeEval: &tfv2.TFReplicaSpec{
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
		err := ValidateAlphaTwoTFJobSpec(&c)
		if err == nil {
			t.Error("Expected error got nil")
		}
	}
}

func TestValidateBetaOneTFJobSpec(t *testing.T) {
	testCases := []tfv1beta1.TFJobSpec{
		{
			TFReplicaSpecs: nil,
		},
		{
			TFReplicaSpecs: map[tfv1beta1.TFReplicaType]*common.ReplicaSpec{
				tfv1beta1.TFReplicaTypeWorker: &common.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[tfv1beta1.TFReplicaType]*common.ReplicaSpec{
				tfv1beta1.TFReplicaTypeWorker: &common.ReplicaSpec{
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
			TFReplicaSpecs: map[tfv1beta1.TFReplicaType]*common.ReplicaSpec{
				tfv1beta1.TFReplicaTypeWorker: &common.ReplicaSpec{
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
			TFReplicaSpecs: map[tfv1beta1.TFReplicaType]*common.ReplicaSpec{
				tfv1beta1.TFReplicaTypeChief: &common.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
				tfv1beta1.TFReplicaTypeMaster: &common.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[tfv1beta1.TFReplicaType]*common.ReplicaSpec{
				tfv1beta1.TFReplicaTypeEval: &common.ReplicaSpec{
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
		err := ValidateBetaOneTFJobSpec(&c)
		if err == nil {
			t.Error("Expected error got nil")
		}
	}
}
