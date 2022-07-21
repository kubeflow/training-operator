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

func expectedMXNetJob(cleanPodPolicy commonv1.CleanPodPolicy, restartPolicy commonv1.RestartPolicy, replicas int32, portName string, port int32) *MXJob {
	ports := []corev1.ContainerPort{}

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
	if portName != MXJobDefaultPortName {
		ports = append(ports,
			corev1.ContainerPort{
				Name:          MXJobDefaultPortName,
				ContainerPort: MXJobDefaultPort,
			},
		)
	}

	return &MXJob{
		Spec: MXJobSpec{
			RunPolicy: commonv1.RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,
			},
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
					Replicas:      pointer.Int32(replicas),
					RestartPolicy: restartPolicy,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name:  MXJobDefaultContainerName,
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

func TestSetDefaults_MXJob(t *testing.T) {
	testCases := map[string]struct {
		original *MXJob
		expected *MXJob
	}{
		"set spec with minimum setting": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  MXJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, MXJobDefaultRestartPolicy, 1, MXJobDefaultPortName, MXJobDefaultPort),
		},
		"Set spec with restart policy": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							RestartPolicy: commonv1.RestartPolicyOnFailure,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  MXJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, commonv1.RestartPolicyOnFailure, 1, MXJobDefaultPortName, MXJobDefaultPort),
		},
		"Set spec with replicas": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							Replicas: pointer.Int32(3),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  MXJobDefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, MXJobDefaultRestartPolicy, 3, MXJobDefaultPortName, MXJobDefaultPort),
		},

		"Set spec with default node port name and port": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  MXJobDefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          MXJobDefaultPortName,
													ContainerPort: MXJobDefaultPort,
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
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, MXJobDefaultRestartPolicy, 1, MXJobDefaultPortName, MXJobDefaultPort),
		},

		"Set spec with node port": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXJobReplicaTypeWorker: &commonv1.ReplicaSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name:  MXJobDefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												corev1.ContainerPort{
													Name:          MXJobDefaultPortName,
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
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, MXJobDefaultRestartPolicy, 1, MXJobDefaultPortName, 9999),
		},
	}

	for name, tc := range testCases {
		SetDefaults_MXJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, pformat(tc.expected), pformat(tc.original))
		}
	}

}
