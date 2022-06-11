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

package validation

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	trainingv1 "github.com/kubeflow/training-operator/pkg/apis/training/v1"

	v1 "k8s.io/api/core/v1"
)

func TestValidateV1MpiJobSpec(t *testing.T) {
	testCases := []trainingv1.MPIJobSpec{
		{
			MPIReplicaSpecs: nil,
		},
		{
			MPIReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.MPIReplicaTypeLauncher: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			MPIReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.MPIReplicaTypeLauncher: &commonv1.ReplicaSpec{
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
			MPIReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.MPIReplicaTypeLauncher: &commonv1.ReplicaSpec{
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
			MPIReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.MPIReplicaTypeLauncher: &commonv1.ReplicaSpec{
					Replicas: trainingv1.Int32(2),
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
				},
			},
		},
	}
	for _, c := range testCases {
		err := ValidateV1MpiJobSpec(&c)
		if err == nil {
			t.Error("Failed validate the v1.MpiJobSpec")
		}
	}
}

func TestValidateV1MXJobSpec(t *testing.T) {
	testCases := []trainingv1.MXJobSpec{
		{
			MXReplicaSpecs: nil,
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.MXReplicaTypeWorker: &commonv1.ReplicaSpec{
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
				trainingv1.MXReplicaTypeWorker: &commonv1.ReplicaSpec{
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
				trainingv1.MXReplicaTypeWorker: &commonv1.ReplicaSpec{
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
				trainingv1.MXReplicaTypeScheduler: &commonv1.ReplicaSpec{
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

func TestValidateV1PyTorchJobSpec(t *testing.T) {
	testCases := []trainingv1.PyTorchJobSpec{
		{
			PyTorchReplicaSpecs: nil,
		},
		{
			PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.PyTorchReplicaTypeWorker: {
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
				trainingv1.PyTorchReplicaTypeWorker: {
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
				trainingv1.PyTorchReplicaTypeWorker: {
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
				trainingv1.PyTorchReplicaTypeMaster: {
					Replicas: trainingv1.Int32(2),
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

func TestValidateV1TFJobSpec(t *testing.T) {
	testCases := []trainingv1.TFJobSpec{
		{
			TFReplicaSpecs: nil,
		},
		{
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.TFReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.TFReplicaTypeWorker: &commonv1.ReplicaSpec{
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
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.TFReplicaTypeWorker: &commonv1.ReplicaSpec{
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
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.TFReplicaTypeChief: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
				trainingv1.TFReplicaTypeMaster: &commonv1.ReplicaSpec{
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
		err := ValidateV1TFJobSpec(&c)
		if err == nil {
			t.Error("Expected error got nil")
		}
	}
}

func TestValidateV1XGBoostJobSpec(t *testing.T) {
	testCases := []trainingv1.XGBoostJobSpec{
		{
			XGBReplicaSpecs: nil,
		},
		{
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.XGBoostReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.XGBoostReplicaTypeWorker: &commonv1.ReplicaSpec{
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
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				trainingv1.XGBoostReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
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
				trainingv1.XGBoostReplicaTypeMaster: &commonv1.ReplicaSpec{
					Replicas: trainingv1.Int32(2),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
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
				trainingv1.XGBoostReplicaTypeWorker: &commonv1.ReplicaSpec{
					Replicas: trainingv1.Int32(1),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
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
		err := ValidateV1XGBoostJobSpec(&c)
		if err == nil {
			t.Error("Failed validate the v1.XGBoostJobSpec")
		}
	}
}
