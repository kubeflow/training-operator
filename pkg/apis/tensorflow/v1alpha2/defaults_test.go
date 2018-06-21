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

package v1alpha2

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"

	"github.com/kubeflow/tf-operator/pkg/util"
)

const (
	testImage = "test-image:latest"
)

func expectedTFJob(cleanPodPolicy CleanPodPolicy, restartPolicy RestartPolicy, portName string, port int32) *TFJob {
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

	return &TFJob{
		Spec: TFJobSpec{
			CleanPodPolicy: &cleanPodPolicy,
			TFReplicaSpecs: map[TFReplicaType]*TFReplicaSpec{
				TFReplicaTypeWorker: &TFReplicaSpec{
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
	spec := &TFReplicaSpec{
		RestartPolicy: RestartPolicyAlways,
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

	workerUpperCase := TFReplicaType("WORKER")
	original := &TFJob{
		Spec: TFJobSpec{
			TFReplicaSpecs: map[TFReplicaType]*TFReplicaSpec{
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
	customRestartPolicy := RestartPolicyAlways

	testCases := map[string]struct {
		original *TFJob
		expected *TFJob
	}{
		"set replicas": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[TFReplicaType]*TFReplicaSpec{
						TFReplicaTypeWorker: &TFReplicaSpec{
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
			expected: expectedTFJob(CleanPodPolicyRunning, customRestartPolicy, DefaultPortName, DefaultPort),
		},
		"set replicas with default restartpolicy": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[TFReplicaType]*TFReplicaSpec{
						TFReplicaTypeWorker: &TFReplicaSpec{
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
			expected: expectedTFJob(CleanPodPolicyRunning, DefaultRestartPolicy, DefaultPortName, DefaultPort),
		},
		"set replicas with default port": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[TFReplicaType]*TFReplicaSpec{
						TFReplicaTypeWorker: &TFReplicaSpec{
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
			expected: expectedTFJob(CleanPodPolicyRunning, customRestartPolicy, "", 0),
		},
		"set replicas adding default port": {
			original: &TFJob{
				Spec: TFJobSpec{
					TFReplicaSpecs: map[TFReplicaType]*TFReplicaSpec{
						TFReplicaTypeWorker: &TFReplicaSpec{
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
			expected: expectedTFJob(CleanPodPolicyRunning, customRestartPolicy, customPortName, customPort),
		},
		"set custom cleanpod policy": {
			original: &TFJob{
				Spec: TFJobSpec{
					CleanPodPolicy: cleanPodPolicyPointer(CleanPodPolicyAll),
					TFReplicaSpecs: map[TFReplicaType]*TFReplicaSpec{
						TFReplicaTypeWorker: &TFReplicaSpec{
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
			expected: expectedTFJob(CleanPodPolicyAll, customRestartPolicy, customPortName, customPort),
		},
	}

	for name, tc := range testCases {
		SetDefaults_TFJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, util.Pformat(tc.expected), util.Pformat(tc.original))
		}
	}
}

func cleanPodPolicyPointer(cleanPodPolicy CleanPodPolicy) *CleanPodPolicy {
	c := cleanPodPolicy
	return &c
}
