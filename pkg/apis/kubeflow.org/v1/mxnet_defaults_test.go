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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func expectedMXNetJob(cleanPodPolicy CleanPodPolicy, restartPolicy RestartPolicy, replicas int32, portName string, port int32) *MXJob {
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
			RunPolicy: RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,
			},
			MXReplicaSpecs: map[ReplicaType]*ReplicaSpec{
				MXJobReplicaTypeWorker: {
					Replicas:      ptr.To[int32](replicas),
					RestartPolicy: restartPolicy,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
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
					MXReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MXJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
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
			expected: expectedMXNetJob(CleanPodPolicyNone, MXJobDefaultRestartPolicy, 1, MXJobDefaultPortName, MXJobDefaultPort),
		},
		"Set spec with restart policy": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MXJobReplicaTypeWorker: {
							RestartPolicy: RestartPolicyOnFailure,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
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
			expected: expectedMXNetJob(CleanPodPolicyNone, RestartPolicyOnFailure, 1, MXJobDefaultPortName, MXJobDefaultPort),
		},
		"Set spec with replicas": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MXJobReplicaTypeWorker: {
							Replicas: ptr.To[int32](3),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
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
			expected: expectedMXNetJob(CleanPodPolicyNone, MXJobDefaultRestartPolicy, 3, MXJobDefaultPortName, MXJobDefaultPort),
		},

		"Set spec with default node port name and port": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MXJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  MXJobDefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												{
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
			expected: expectedMXNetJob(CleanPodPolicyNone, MXJobDefaultRestartPolicy, 1, MXJobDefaultPortName, MXJobDefaultPort),
		},

		"Set spec with node port": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MXJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  MXJobDefaultContainerName,
											Image: testImage,
											Ports: []corev1.ContainerPort{
												{
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
			expected: expectedMXNetJob(CleanPodPolicyNone, MXJobDefaultRestartPolicy, 1, MXJobDefaultPortName, 9999),
		},

		"set spec with cleanpod policy": {
			original: &MXJob{
				Spec: MXJobSpec{
					RunPolicy: RunPolicy{
						CleanPodPolicy: CleanPodPolicyPointer(CleanPodPolicyAll),
					},
					MXReplicaSpecs: map[ReplicaType]*ReplicaSpec{
						MXJobReplicaTypeWorker: {
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
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
			expected: expectedMXNetJob(CleanPodPolicyAll, MXJobDefaultRestartPolicy, 1, MXJobDefaultPortName, MXJobDefaultPort),
		},
	}

	for name, tc := range testCases {
		SetDefaults_MXJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, pformat(tc.expected), pformat(tc.original))
		}
	}

}
