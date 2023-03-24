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

func TestValidateV1TFJob(t *testing.T) {
	validTFReplicaSpecs := map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
		TFJobReplicaTypeWorker: {
			Replicas:      pointer.Int32(2),
			RestartPolicy: commonv1.RestartPolicyOnFailure,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "tensorflow",
						Image: "kubeflow/tf-mnist-with-summaries:latest",
						Command: []string{
							"python",
							"/var/tf_mnist/mnist_with_summaries.py",
						},
					}},
				},
			},
		},
	}

	testCases := map[string]struct {
		tfJob   *TFJob
		wantErr bool
	}{
		"valid tfJob": {
			tfJob: &TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: TFJobSpec{
					TFReplicaSpecs: validTFReplicaSpecs,
				},
			},
			wantErr: false,
		},
		"TFJob name does not meet DNS1035": {
			tfJob: &TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "00test",
				},
				Spec: TFJobSpec{
					TFReplicaSpecs: validTFReplicaSpecs,
				},
			},
			wantErr: true,
		},
		"no containers": {
			tfJob: &TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: TFJobSpec{
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFJobReplicaTypeWorker: {
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
		"empty image": {
			tfJob: &TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: TFJobSpec{
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:  "tensorflow",
										Image: "",
									}},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		"tfJob default container name doesn't present": {
			tfJob: &TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: TFJobSpec{
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:  "",
										Image: "kubeflow/tf-dist-mnist-test:1.0",
									}},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		"there are more than 2 masterReplica's or ChiefReplica's": {
			tfJob: &TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: TFJobSpec{
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFJobReplicaTypeChief: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{},
								},
							},
						},
						TFJobReplicaTypeMaster: {
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
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := ValidateV1TFJob(tc.tfJob)
			if (got != nil) != tc.wantErr {
				t.Fatalf("ValidateV1TFJob() error = %v, wantErr %v", got, tc.wantErr)
			}
		})
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
