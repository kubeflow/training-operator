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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestValidateV1PyTorchJob(t *testing.T) {
	validPyTorchReplicaSpecs := map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
		PyTorchJobReplicaTypeMaster: {
			Replicas:      pointer.Int32(1),
			RestartPolicy: commonv1.RestartPolicyOnFailure,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "pytorch",
						Image:           "docker.io/kubeflowkatib/pytorch-mnist:v1beta1-45c5727",
						ImagePullPolicy: corev1.PullAlways,
						Command: []string{
							"python3",
							"/opt/pytorch-mnist/mnist.py",
							"--epochs=1",
						},
					}},
				},
			},
		},
		PyTorchJobReplicaTypeWorker: {
			Replicas:      pointer.Int32(1),
			RestartPolicy: commonv1.RestartPolicyOnFailure,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "pytorch",
						Image:           "docker.io/kubeflowkatib/pytorch-mnist:v1beta1-45c5727",
						ImagePullPolicy: corev1.PullAlways,
						Command: []string{
							"python3",
							"/opt/pytorch-mnist/mnist.py",
							"--epochs=1",
						},
					}},
				},
			},
		},
	}

	testCases := map[string]struct {
		pytorchJob *PyTorchJob
		wantErr    bool
	}{
		"valid PyTorchJob": {
			pytorchJob: &PyTorchJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PyTorchJobSpec{
					PyTorchReplicaSpecs: validPyTorchReplicaSpecs,
				},
			},
			wantErr: false,
		},
		"pytorchJob name does not meet DNS1035": {
			pytorchJob: &PyTorchJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "0-test",
				},
				Spec: PyTorchJobSpec{
					PyTorchReplicaSpecs: validPyTorchReplicaSpecs,
				},
			},
			wantErr: true,
		},
		"no containers": {
			pytorchJob: &PyTorchJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PyTorchJobSpec{
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						PyTorchJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		"image is empty": {
			pytorchJob: &PyTorchJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PyTorchJobSpec{
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						PyTorchJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "pytorch",
											Image: "",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		"pytorchJob default container name doesn't present": {
			pytorchJob: &PyTorchJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PyTorchJobSpec{
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						PyTorchJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
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
			},
			wantErr: true,
		},
		"the number of replicas in masterReplica is other than 1": {
			pytorchJob: &PyTorchJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PyTorchJobSpec{
					PyTorchReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						PyTorchJobReplicaTypeMaster: {
							Replicas: pointer.Int32(2),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
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
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := ValidateV1PyTorchJob(tc.pytorchJob)
			if (got != nil) != tc.wantErr {
				t.Fatalf("ValidateV1PyTorchJob() error = %v, wantErr %v", got, tc.wantErr)
			}
		})
	}
}
