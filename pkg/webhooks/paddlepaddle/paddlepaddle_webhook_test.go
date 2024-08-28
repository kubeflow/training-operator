/*
Copyright 2024 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package paddlepaddle

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/common/util"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

func TestValidateV1PaddleJob(t *testing.T) {
	validPaddleReplicaSpecs := map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
		trainingoperator.PaddleJobReplicaTypeWorker: {
			Replicas:      ptr.To[int32](2),
			RestartPolicy: trainingoperator.RestartPolicyNever,
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
		paddleJob *trainingoperator.PaddleJob
		wantErr   field.ErrorList
	}{
		"valid paddleJob": {
			paddleJob: &trainingoperator.PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.PaddleJobSpec{
					RunPolicy: trainingoperator.RunPolicy{
						ManagedBy: ptr.To(trainingoperator.KubeflowJobsController),
					},
					PaddleReplicaSpecs: validPaddleReplicaSpecs,
				},
			},
		},
		"paddleJob name does not meet DNS1035": {
			paddleJob: &trainingoperator.PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "__test",
				},
				Spec: trainingoperator.PaddleJobSpec{
					PaddleReplicaSpecs: validPaddleReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("metadata").Child("name"), "", ""),
			},
		},
		"no containers": {
			paddleJob: &trainingoperator.PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.PaddleJobSpec{
					PaddleReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.PaddleJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{},
								},
							},
						},
					},
				},
			},
			wantErr: field.ErrorList{
				field.Required(paddleReplicaSpecPath.Key(string(trainingoperator.PaddleJobReplicaTypeWorker)).Child("template").Child("spec").Child("containers"), ""),
				field.Required(paddleReplicaSpecPath.Key(string(trainingoperator.PaddleJobReplicaTypeWorker)).Child("template").Child("spec").Child("containers"), ""),
			},
		},
		"image is empty": {
			paddleJob: &trainingoperator.PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.PaddleJobSpec{
					PaddleReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.PaddleJobReplicaTypeWorker: {
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
			wantErr: field.ErrorList{
				field.Required(paddleReplicaSpecPath.Key(string(trainingoperator.PaddleJobReplicaTypeWorker)).Child("template").Child("spec").Child("containers").Index(0).Child("image"), ""),
			},
		},
		"paddle default container name doesn't find": {
			paddleJob: &trainingoperator.PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.PaddleJobSpec{
					PaddleReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.PaddleJobReplicaTypeWorker: {
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
			wantErr: field.ErrorList{
				field.Required(paddleReplicaSpecPath.Key(string(trainingoperator.PaddleJobReplicaTypeWorker)).Child("template").Child("spec").Child("containers"), ""),
			},
		},
		"replicaSpec is nil": {
			paddleJob: &trainingoperator.PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.PaddleJobSpec{
					PaddleReplicaSpecs: nil,
				},
			},
			wantErr: field.ErrorList{
				field.Required(paddleReplicaSpecPath, ""),
			},
		},
		"managedBy controller name is malformed": {
			paddleJob: &trainingoperator.PaddleJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.PaddleJobSpec{
					RunPolicy: trainingoperator.RunPolicy{
						ManagedBy: ptr.To(testutil.MalformedManagedBy),
					},
					PaddleReplicaSpecs: validPaddleReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("spec").Child("managedBy"), "", ""),
				field.NotSupported(field.NewPath("spec").Child("managedBy"), "", sets.List(util.SupportedJobControllers)),
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := validatePaddleJob(tc.paddleJob)
			if diff := cmp.Diff(tc.wantErr, got, cmpopts.IgnoreFields(field.Error{}, "Detail", "BadValue")); len(diff) != 0 {
				t.Errorf("Unexpected error (-want,+got):\n%s", diff)
			}
		})
	}
}
