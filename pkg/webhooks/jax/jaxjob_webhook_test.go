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

package jax

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func TestValidateV1JAXJob(t *testing.T) {
	validJAXReplicaSpecs := map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
		trainingoperator.JAXJobReplicaTypeWorker: {
			Replicas:      ptr.To[int32](1),
			RestartPolicy: trainingoperator.RestartPolicyOnFailure,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "jax",
						Image: "docker.io/kubeflow/jaxjob-simple:latest",
						Ports: []corev1.ContainerPort{{
							Name:          "jaxjob-port",
							ContainerPort: 6666,
						}},
						ImagePullPolicy: corev1.PullAlways,
						Command: []string{
							"python",
							"train.py",
						},
					}},
				},
			},
		},
	}

	testCases := map[string]struct {
		jaxJob  *trainingoperator.JAXJob
		wantErr field.ErrorList
	}{
		"valid JAXJob": {
			jaxJob: &trainingoperator.JAXJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.JAXJobSpec{
					JAXReplicaSpecs: validJAXReplicaSpecs,
				},
			},
		},
		"jaxJob name does not meet DNS1035": {
			jaxJob: &trainingoperator.JAXJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "0-test",
				},
				Spec: trainingoperator.JAXJobSpec{
					JAXReplicaSpecs: validJAXReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("metadata").Child("name"), "", ""),
			},
		},
		"no containers": {
			jaxJob: &trainingoperator.JAXJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.JAXJobSpec{
					JAXReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.JAXJobReplicaTypeWorker: {
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
				field.Required(jaxReplicaSpecPath.
					Key(string(trainingoperator.JAXJobReplicaTypeWorker)).
					Child("template").
					Child("spec").
					Child("containers"), ""),
				field.Required(jaxReplicaSpecPath.
					Key(string(trainingoperator.JAXJobReplicaTypeWorker)).
					Child("template").
					Child("spec").
					Child("containers"), ""),
			},
		},
		"image is empty": {
			jaxJob: &trainingoperator.JAXJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.JAXJobSpec{
					JAXReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.JAXJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "jax",
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
				field.Required(jaxReplicaSpecPath.
					Key(string(trainingoperator.JAXJobReplicaTypeWorker)).
					Child("template").
					Child("spec").
					Child("containers").
					Index(0).
					Child("image"), ""),
			},
		},
		"jaxJob default container name doesn't present": {
			jaxJob: &trainingoperator.JAXJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.JAXJobSpec{
					JAXReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.JAXJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "",
											Image: "gcr.io/kubeflow-ci/jaxjob-dist-spmd-mnist_test:1.0",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: field.ErrorList{
				field.Required(jaxReplicaSpecPath.
					Key(string(trainingoperator.JAXJobReplicaTypeWorker)).
					Child("template").
					Child("spec").
					Child("containers"), ""),
			},
		},
		"replicaSpec is nil": {
			jaxJob: &trainingoperator.JAXJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.JAXJobSpec{
					JAXReplicaSpecs: nil,
				},
			},
			wantErr: field.ErrorList{
				field.Required(jaxReplicaSpecPath, ""),
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := validateJAXJob(tc.jaxJob)
			if diff := cmp.Diff(tc.wantErr, got, cmpopts.IgnoreFields(field.Error{}, "Detail", "BadValue")); len(diff) != 0 {
				t.Errorf("Unexpected error (-want,+got):\n%s", diff)
			}
		})
	}
}
