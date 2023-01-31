// Copyright 2022 The Kubeflow Authors
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

func TestValidateV1PaddleJob(t *testing.T) {
	validPaddleReplicaSpecs := map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
		PaddleJobReplicaTypeWorker: {
			Replicas:      pointer.Int32(2),
			RestartPolicy: commonv1.RestartPolicyNever,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "paddle",
						Image:   "registry.baidubce.com/paddlepaddle/paddle:2.4.0rc0-cpu",
						Command: []string{"python"},
						Args: []string{
							"-m",
							"paddle.distributed.launch",
							"run_check",
						},
						Ports: []corev1.ContainerPort{{
							Name:          "master",
							ContainerPort: int32(37777),
						}},
						ImagePullPolicy: corev1.PullAlways,
					}},
				},
			},
		},
	}

	testCases := map[string]struct {
		paddleJob *PaddleJob
		wantErr   bool
	}{
		"valid paddleJob": {
			paddleJob: &PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PaddleJobSpec{
					PaddleReplicaSpecs: validPaddleReplicaSpecs,
				},
			},
			wantErr: false,
		},
		"paddleJob name does not meet DNS1035": {
			paddleJob: &PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "__test",
				},
				Spec: PaddleJobSpec{
					PaddleReplicaSpecs: validPaddleReplicaSpecs,
				},
			},
			wantErr: true,
		},
		"no containers": {
			paddleJob: &PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PaddleJobSpec{
					PaddleReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						PaddleJobReplicaTypeWorker: {
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
			paddleJob: &PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PaddleJobSpec{
					PaddleReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						PaddleJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "paddle",
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
		"paddle default container name doesn't find": {
			paddleJob: &PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PaddleJobSpec{
					PaddleReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						PaddleJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "",
											Image: "gcr.io/kubeflow-ci/paddle-dist-mnist_test:1.0",
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
		"replicaSpec is nil": {
			paddleJob: &PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: PaddleJobSpec{
					PaddleReplicaSpecs: nil,
				},
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := ValidateV1PaddleJob(tc.paddleJob)
			if (got != nil) != tc.wantErr {
				t.Fatalf("ValidateV1PaddleJob() error = %v, wantErr %v", got, tc.wantErr)
			}
		})
	}
}
