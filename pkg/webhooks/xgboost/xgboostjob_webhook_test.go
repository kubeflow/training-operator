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

package xgboost

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

func TestValidateXGBoostJob(t *testing.T) {
	validXGBoostReplicaSpecs := map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
		trainingoperator.XGBoostJobReplicaTypeMaster: {
			Replicas:      ptr.To[int32](1),
			RestartPolicy: trainingoperator.RestartPolicyNever,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "xgboost",
						Image: "docker.io/kubeflow/xgboost-dist-iris:latest",
						Ports: []corev1.ContainerPort{{
							Name:          "xgboostjob-port",
							ContainerPort: 9991,
						}},
						ImagePullPolicy: corev1.PullAlways,
						Args: []string{
							"--job_type=Train",
							"--xgboost_parameter=objective:multi:softprob,num_class:3",
							"--n_estimators=10",
							"--learning_rate=0.1",
							"--model_path=/tmp/xgboost-model",
							"--model_storage_type=local",
						},
					}},
				},
			},
		},
		trainingoperator.XGBoostJobReplicaTypeWorker: {
			Replicas:      ptr.To[int32](2),
			RestartPolicy: trainingoperator.RestartPolicyExitCode,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "xgboost",
						Image: "docker.io/kubeflow/xgboost-dist-iris:latest",
						Ports: []corev1.ContainerPort{{
							Name:          "xgboostjob-port",
							ContainerPort: 9991,
						}},
						ImagePullPolicy: corev1.PullAlways,
						Args: []string{
							"--job_type=Train",
							"--xgboost_parameter=objective:multi:softprob,num_class:3",
							"--n_estimators=10",
							"--learning_rate=0.1",
						},
					}},
				},
			},
		},
	}

	testCases := map[string]struct {
		xgboostJob *trainingoperator.XGBoostJob
		wantErr    field.ErrorList
	}{
		"valid XGBoostJob": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					RunPolicy: trainingoperator.RunPolicy{
						ManagedBy: ptr.To(trainingoperator.KubeflowJobsController),
					},
					XGBReplicaSpecs: validXGBoostReplicaSpecs,
				},
			},
		},
		"XGBoostJob name does not meet DNS1035": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "-test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					XGBReplicaSpecs: validXGBoostReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("metadata").Child("name"), "", ""),
			},
		},
		"empty containers": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					XGBReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.XGBoostJobReplicaTypeMaster: {
							Replicas: ptr.To[int32](1),
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
				field.Required(xgbReplicaSpecPath.Key(string(trainingoperator.XGBoostJobReplicaTypeMaster)).Child("template").Child("spec").Child("containers"), ""),
				field.Required(xgbReplicaSpecPath.Key(string(trainingoperator.XGBoostJobReplicaTypeMaster)).Child("template").Child("spec").Child("containers"), ""),
			},
		},
		"image is empty": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					XGBReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.XGBoostJobReplicaTypeMaster: {
							Replicas: ptr.To[int32](1),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:  "xgboost",
										Image: "",
									}},
								},
							},
						},
					},
				},
			},
			wantErr: field.ErrorList{
				field.Required(xgbReplicaSpecPath.Key(string(trainingoperator.XGBoostJobReplicaTypeMaster)).Child("template").Child("spec").Child("containers").Index(0).Child("image"), ""),
			},
		},
		"xgboostJob default container name doesn't present": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					XGBReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.XGBoostJobReplicaTypeMaster: {
							Replicas: ptr.To[int32](1),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:  "",
										Image: "gcr.io/kubeflow-ci/xgboost-dist-mnist_test:1.0",
									}},
								},
							},
						},
					},
				},
			},
			wantErr: field.ErrorList{
				field.Required(xgbReplicaSpecPath.Key(string(trainingoperator.XGBoostJobReplicaTypeMaster)).Child("template").Child("spec").Child("containers"), ""),
			},
		},
		"the number of replicas in masterReplica is other than 1": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					XGBReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.XGBoostJobReplicaTypeMaster: {
							Replicas: ptr.To[int32](2),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:  "xgboost",
										Image: "gcr.io/kubeflow-ci/xgboost-dist-mnist_test:1.0",
									}},
								},
							},
						},
					},
				},
			},
			wantErr: field.ErrorList{
				field.Forbidden(xgbReplicaSpecPath.Key(string(trainingoperator.XGBoostJobReplicaTypeMaster)).Child("replicas"), ""),
			},
		},
		"masterReplica does not exist": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					XGBReplicaSpecs: map[trainingoperator.ReplicaType]*trainingoperator.ReplicaSpec{
						trainingoperator.XGBoostJobReplicaTypeWorker: {
							Replicas: ptr.To[int32](1),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:  "xgboost",
										Image: "gcr.io/kubeflow-ci/xgboost-dist-mnist_test:1.0",
									}},
								},
							},
						},
					},
				},
			},
			wantErr: field.ErrorList{
				field.Required(xgbReplicaSpecPath.Key(string(trainingoperator.XGBoostJobReplicaTypeMaster)), ""),
			},
		},
		"managedBy controller name is malformed": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					RunPolicy: trainingoperator.RunPolicy{
						ManagedBy: ptr.To(testutil.MalformedManagedBy),
					},
					XGBReplicaSpecs: validXGBoostReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("spec").Child("managedBy"), "", ""),
			},
		},
		"managedBy controller name is too long": {
			xgboostJob: &trainingoperator.XGBoostJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: trainingoperator.XGBoostJobSpec{
					RunPolicy: trainingoperator.RunPolicy{
						ManagedBy: ptr.To(testutil.TooLongManagedBy),
					},
					XGBReplicaSpecs: validXGBoostReplicaSpecs,
				},
			},
			wantErr: field.ErrorList{
				field.TooLongMaxLength(field.NewPath("spec").Child("managedBy"), "", trainingoperator.MaxManagedByLength),
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := validateXGBoostJob(tc.xgboostJob)
			if diff := cmp.Diff(tc.wantErr, got, cmpopts.IgnoreFields(field.Error{}, "Detail", "BadValue")); len(diff) != 0 {
				t.Errorf("Unexpected errors (-want,+got):\n%s", diff)
			}
		})
	}
}
