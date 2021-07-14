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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

const (
	testImage = "test-image:latest"
)

func expectedMXNetJob(cleanPodPolicy commonv1.CleanPodPolicy, restartPolicy commonv1.RestartPolicy, replicas int32, portName string, port int32) *MXJob {
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

	return &MXJob{
		Spec: MXJobSpec{
			RunPolicy: commonv1.RunPolicy{
				CleanPodPolicy: &cleanPodPolicy,
			},
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				MXReplicaTypeWorker: &commonv1.ReplicaSpec{
					Replicas:      Int32(replicas),
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

func TestSetDefaults_MXJob(t *testing.T) {
	testCases := map[string]struct {
		original *MXJob
		expected *MXJob
	}{
		"set spec with minimum setting": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXReplicaTypeWorker: &commonv1.ReplicaSpec{
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
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, DefaultRestartPolicy, 1, DefaultPortName, DefaultPort),
		},
		"Set spec with restart policy": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXReplicaTypeWorker: &commonv1.ReplicaSpec{
							RestartPolicy: commonv1.RestartPolicyOnFailure,
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
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, commonv1.RestartPolicyOnFailure, 1, DefaultPortName, DefaultPort),
		},
		"Set spec with replicas": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXReplicaTypeWorker: &commonv1.ReplicaSpec{
							Replicas: Int32(3),
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
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, DefaultRestartPolicy, 3, DefaultPortName, DefaultPort),
		},

		"Set spec with default node port name and port": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXReplicaTypeWorker: &commonv1.ReplicaSpec{
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
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, DefaultRestartPolicy, 1, DefaultPortName, DefaultPort),
		},

		"Set spec with node port": {
			original: &MXJob{
				Spec: MXJobSpec{
					MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
						MXReplicaTypeWorker: &commonv1.ReplicaSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										v1.Container{
											Name:  DefaultContainerName,
											Image: testImage,
											Ports: []v1.ContainerPort{
												v1.ContainerPort{
													Name:          DefaultPortName,
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
			expected: expectedMXNetJob(commonv1.CleanPodPolicyAll, DefaultRestartPolicy, 1, DefaultPortName, 9999),
		},
	}

	for name, tc := range testCases {
		SetDefaults_MXJob(tc.original)
		if !reflect.DeepEqual(tc.original, tc.expected) {
			t.Errorf("%s: Want\n%v; Got\n %v", name, pformat(tc.expected), pformat(tc.original))
		}
	}

}

// pformat returns a pretty format output of any value that can be marshaled to JSON.
func pformat(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	valueJSON, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Warningf("Couldn't pretty format %v, error: %v", value, err)
		return fmt.Sprintf("%v", value)
	}
	return string(valueJSON)
}
