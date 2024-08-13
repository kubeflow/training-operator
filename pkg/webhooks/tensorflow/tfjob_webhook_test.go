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

package tensorflow

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	trainingoperator "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/util/testutil"
)

func TestValidateTFJob(t *testing.T) {
	validTFReplicaSpecs := map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
		trainingoperator.TFJobReplicaTypeWorker: {
			Replicas:      ptr.To[int32](2),
			RestartPolicy: trainingoperator.RestartPolicyOnFailure,
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
		tfJob   *trainingoperator.TFJob
		wantErr field.ErrorList
	}{
		"valid tfJob": {
			tfJob: &trainingoperator.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.TFJobSpec{
					RunPolicy: trainingoperator.RunPolicy{
						ManagedBy: ptr.To(trainingoperator.KubeflowJobsController),
					},
					TFReplicaSpecs: validTFReplicaSpecs,
				},
			},
		},
		"TFJob name does not meet DNS1035": {
			tfJob: &trainingoperator.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "00test",
				},
				Spec: trainingoperator.TFJobSpec{
					TFReplicaSpecs: validTFReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("metadata").Child("name"), "", ""),
			},
		},
		"no containers": {
			tfJob: &trainingoperator.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.TFJobSpec{
					TFReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.TFJobReplicaTypeWorker: {
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
				field.Required(tfReplicaSpecPath.Key(string(trainingoperator.TFJobReplicaTypeWorker)).Child("template").Child("spec").Child("containers"), ""),
				field.Required(tfReplicaSpecPath.Key(string(trainingoperator.TFJobReplicaTypeWorker)).Child("template").Child("spec").Child("containers"), ""),
			},
		},
		"empty image": {
			tfJob: &trainingoperator.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.TFJobSpec{
					TFReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.TFJobReplicaTypeWorker: {
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
			wantErr: field.ErrorList{
				field.Required(tfReplicaSpecPath.Key(string(trainingoperator.TFJobReplicaTypeWorker)).Child("template").Child("spec").Child("containers").Index(0).Child("image"), ""),
			},
		},
		"tfJob default container name doesn't present": {
			tfJob: &trainingoperator.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.TFJobSpec{
					TFReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.TFJobReplicaTypeWorker: {
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
			wantErr: field.ErrorList{
				field.Required(tfReplicaSpecPath.Key(string(trainingoperator.TFJobReplicaTypeWorker)).Child("template").Child("spec").Child("containers"), ""),
			},
		},
		"there are more than 2 masterReplica's or ChiefReplica's": {
			tfJob: &trainingoperator.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.TFJobSpec{
					TFReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.TFJobReplicaTypeChief: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:  "tensorflow",
										Image: "kubeflow/tf-dist-mnist-test:1.0",
									}},
								},
							},
						},
						trainingoperator.TFJobReplicaTypeMaster: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:  "tensorflow",
										Image: "kubeflow/tf-dist-mnist-test:1.0",
									}},
								},
							},
						},
					},
				},
			},
			wantErr: field.ErrorList{
				field.Forbidden(tfReplicaSpecPath, ""),
			},
		},
		"managedBy controller name is malformed": {
			tfJob: &trainingoperator.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.TFJobSpec{
					RunPolicy: trainingoperator.RunPolicy{
						ManagedBy: ptr.To(testutil.MalformedManagedBy),
					},
					TFReplicaSpecs: validTFReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("spec").Child("managedBy"), "", ""),
			},
		},
		"managedBy controller name is too long": {
			tfJob: &trainingoperator.TFJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.TFJobSpec{
					RunPolicy: trainingoperator.RunPolicy{
						ManagedBy: ptr.To(testutil.TooLongManagedBy),
					},
					TFReplicaSpecs: validTFReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.TooLongMaxLength(field.NewPath("spec").Child("managedBy"), "", trainingoperator.MaxManagedByLength),
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := validateTFJob(tc.tfJob)
			if diff := cmp.Diff(tc.wantErr, got, cmpopts.IgnoreFields(field.Error{}, "Detail", "BadValue")); len(diff) != 0 {
				t.Errorf("Unexpected error (-want,+got):\n%s", diff)
			}
		})
	}
}
