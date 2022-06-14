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
	"reflect"
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func expectedXGBoostJob(cleanPodPolicy commonv1.CleanPodPolicy, restartPolicy commonv1.RestartPolicy, replicas int32, portName string, port int32) *XGBoostJob {
	var ports []corev1.ContainerPort

	// port not set
	if portName != "" {
		ports = append(ports,
			corev1.ContainerPort{
				Name:          portName,
				ContainerPort: port,
			},
		)
	}

	// port set with custom name
	if portName != XGBoostJobDefaultPortName {
		ports = append(ports,
			corev1.ContainerPort{
				Name:          XGBoostJobDefaultPortName,
				ContainerPort: XGBoostJobDefaultPort,
			},
		)
	}

	return &XGBoostJob{
		Spec: XGBoostJobSpec{
			RunPolicy: commonv1.RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,
			},
			XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Replicas:      pointer.Int32(replicas),
					RestartPolicy: restartPolicy,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name:  XGBoostJobDefaultContainerName,
									Image: testImage,
									Ports: ports,
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestSetDefaults_XGBoostJob(t *testing.T) {
	testCases := map[string]struct {
		original *XGBoostJob
		expected *XGBoostJob
	}{
		"set spec with minimum setting": {
			original: &XGBoostJob{
				Spec: XGBoostJobSpec{
					XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  XGBoostJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedXGBoostJob(commonv1.CleanPodPolicyAll, XGBoostJobDefaultRestartPolicy, 1, XGBoostJobDefaultPortName, XGBoostJobDefaultPort),
		},
		"Set spec with restart policy": {
			original: &XGBoostJob{
				Spec: XGBoostJobSpec{
					XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							RestartPolicy: commonv1.RestartPolicyOnFailure,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  XGBoostJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedXGBoostJob(commonv1.CleanPodPolicyAll, commonv1.RestartPolicyOnFailure, 1, XGBoostJobDefaultPortName, XGBoostJobDefaultPort),
		},
		"Set spec with replicas": {
			original: &XGBoostJob{
				Spec: XGBoostJobSpec{
					XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							Replicas: pointer.Int32(3),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  XGBoostJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedXGBoostJob(commonv1.CleanPodPolicyAll, XGBoostJobDefaultRestartPolicy, 3, XGBoostJobDefaultPortName, XGBoostJobDefaultPort),
		},

		"Set spec with default node port name and port": {
			original: &XGBoostJob{
				Spec: XGBoostJobSpec{
					XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  XGBoostJobDefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          XGBoostJobDefaultPortName,
													ContainerPort: XGBoostJobDefaultPort,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedXGBoostJob(commonv1.CleanPodPolicyAll, XGBoostJobDefaultRestartPolicy, 1, XGBoostJobDefaultPortName, XGBoostJobDefaultPort),
		},

		"Set spec with node port": {
			original: &XGBoostJob{
				Spec: XGBoostJobSpec{
					XGBReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						XGBoostJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  XGBoostJobDefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          XGBoostJobDefaultPortName,
													ContainerPort: 9999,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedXGBoostJob(commonv1.CleanPodPolicyAll, XGBoostJobDefaultRestartPolicy, 1, XGBoostJobDefaultPortName, 9999),
		},
	}

	for name, tc := range testCases {
		SetDefaults_XGBoostJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, pformat(tc.expected), pformat(tc.original))
		}
	}

}
