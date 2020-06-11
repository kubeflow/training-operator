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

	v1 "k8s.io/api/core/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
)

const (
	testImage = "test-image:latest"
)

func expectedTFJob(cleanPodPolicy commonv1.CleanPodPolicy, restartPolicy commonv1.RestartPolicy, portName string, port int32) *TFJob {
	ports := []v1.ContainerPort{}

	// port not set
	if portName != "" {
		ports = append(ports,
			v1.ContainerPort{
				Name:          portName,
				ContainerPort: port,
			},
		)
	}

	// port set with custom name
	if portName != DefaultPortName {
		ports = append(ports,
			v1.ContainerPort{
				Name:          DefaultPortName,
				ContainerPort: DefaultPort,
			},
		)
	}

	defaultSuccessPolicy := SuccessPolicyDefault

	return &TFJob{
		Spec: TFJobSpec{
			SuccessPolicy: &defaultSuccessPolicy,
			RunPolicy: commonv1.RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,
			},
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				TFReplicaTypeWorker: &commonv1.ReplicaSpec{
					Replicas:      Int32(1),
					RestartPolicy: restartPolicy,
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Name:  DefaultContainerName,
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

func TestSetTypeNames(t *testing.T) {
	spec := &commonv1.ReplicaSpec{
		RestartPolicy: commonv1.RestartPolicyAlways,
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Name:  DefaultContainerName,
						Image: testImage,
						Ports: []v1.ContainerPort{
							v1.ContainerPort{
								Name:          DefaultPortName,
								ContainerPort: DefaultPort,
							},
						},
					},
				},
			},
		},
	}

	workerUpperCase := commonv1.ReplicaType("WORKER")
	original := &TFJob{
		Spec: TFJobSpec{
			TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				workerUpperCase: spec,
			},
		},
	}

	setTypeNamesToCamelCase(original)
	if _, ok := original.Spec.TFReplicaSpecs[workerUpperCase]; ok {
		t.Errorf("Failed to delete key %s", workerUpperCase)
	}
	if _, ok := original.Spec.TFReplicaSpecs[TFReplicaTypeWorker]; !ok {
		t.Errorf("Failed to set key %s", TFReplicaTypeWorker)
	}
}

func TestSetDefaultTFJob(t *testing.T) {
	customPortName := "customPort"
	var customPort int32 = 1234
	customRestartPolicy := commonv1.RestartPolicyAlways

	testCases := map[string]struct {
		original *TFJob
		expected *TFJob
	}{
		"set replicas": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFReplicaTypeWorker: &commonv1.ReplicaSpec{
							RestartPolicy: customRestartPolicy,
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										v1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												v1.ContainerPort{
													Name:          DefaultPortName,
													ContainerPort: DefaultPort,
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
			expected: expectedTFJob(commonv1.CleanPodPolicyRunning, customRestartPolicy, DefaultPortName, DefaultPort),
		},
		"set replicas with default restartpolicy": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFReplicaTypeWorker: &commonv1.ReplicaSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										v1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												v1.ContainerPort{
													Name:          DefaultPortName,
													ContainerPort: DefaultPort,
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
			expected: expectedTFJob(commonv1.CleanPodPolicyRunning, DefaultRestartPolicy, DefaultPortName, DefaultPort),
		},
		"set replicas with default port": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFReplicaTypeWorker: &commonv1.ReplicaSpec{
							Replicas:      Int32(1),
							RestartPolicy: customRestartPolicy,
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										v1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: expectedTFJob(commonv1.CleanPodPolicyRunning, customRestartPolicy, "", 0),
		},
		"set replicas adding default port": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFReplicaTypeWorker: &commonv1.ReplicaSpec{
							Replicas:      Int32(1),
							RestartPolicy: customRestartPolicy,
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										v1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												v1.ContainerPort{
													Name:          customPortName,
													ContainerPort: customPort,
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
			expected: expectedTFJob(commonv1.CleanPodPolicyRunning, customRestartPolicy, customPortName, customPort),
		},
		"set custom cleanpod policy": {
			original: &TFJob{
				Spec: TFJobSpec{
					RunPolicy: commonv1.RunPolicy{
						CleanPodPolicy: cleanPodPolicyPointer(commonv1.CleanPodPolicyAll),
					},
					TFReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						TFReplicaTypeWorker: &commonv1.ReplicaSpec{
							Replicas:      Int32(1),
							RestartPolicy: customRestartPolicy,
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										v1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												v1.ContainerPort{
													Name:          customPortName,
													ContainerPort: customPort,
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
			expected: expectedTFJob(commonv1.CleanPodPolicyAll, customRestartPolicy, customPortName, customPort),
		},
	}

	for name, tc := range testCases {
		SetDefaults_TFJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, util.Pformat(tc.expected), util.Pformat(tc.original))
		}
	}
}

func cleanPodPolicyPointer(cleanPodPolicy commonv1.CleanPodPolicy) *commonv1.CleanPodPolicy {
	c := cleanPodPolicy
	return &c
}
