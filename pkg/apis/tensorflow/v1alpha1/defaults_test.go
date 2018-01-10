package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/tensorflow/k8s/pkg/util"
	"k8s.io/api/core/v1"
)

func TestSetDefaults_TfJob(t *testing.T) {
	type testCase struct {
		in       *TfJob
		expected *TfJob
	}

	testCases := []testCase{
		{
			in: &TfJob{
				Spec: TfJobSpec{
					ReplicaSpecs: []*TfReplicaSpec{
						{
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "tensorflow",
										},
									},
								},
							},
						},
					},
					TfImage: "tensorflow/tensorflow:1.3.0",
				},
			},
			expected: &TfJob{
				Spec: TfJobSpec{
					ReplicaSpecs: []*TfReplicaSpec{
						{
							Replicas: proto.Int32(1),
							TfPort:   proto.Int32(2222),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "tensorflow",
										},
									},
								},
							},
							TfReplicaType: MASTER,
						},
					},
					TfImage: "tensorflow/tensorflow:1.3.0",
					TerminationPolicy: &TerminationPolicySpec{
						Chief: &ChiefSpec{
							ReplicaName:  "MASTER",
							ReplicaIndex: 0,
						},
					},
				},
			},
		},
		{
			in: &TfJob{
				Spec: TfJobSpec{
					ReplicaSpecs: []*TfReplicaSpec{
						{
							TfReplicaType: PS,
						},
					},
					TfImage: "tensorflow/tensorflow:1.3.0",
				},
			},
			expected: &TfJob{
				Spec: TfJobSpec{
					ReplicaSpecs: []*TfReplicaSpec{
						{
							Replicas: proto.Int32(1),
							TfPort:   proto.Int32(2222),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										v1.Container{
											Image: "tensorflow/tensorflow:1.3.0",
											Name:  "tensorflow",
											VolumeMounts: []v1.VolumeMount{
												v1.VolumeMount{
													Name:      "ps-config-volume",
													MountPath: "/ps-server",
												},
											},
										},
									},
									RestartPolicy: v1.RestartPolicyOnFailure,
								},
							},
							TfReplicaType: PS,
							IsDefaultPS:   true,
						},
					},
					TfImage: "tensorflow/tensorflow:1.3.0",
					TerminationPolicy: &TerminationPolicySpec{
						Chief: &ChiefSpec{
							ReplicaName:  "MASTER",
							ReplicaIndex: 0,
						},
					},
				},
			},
		},
	}

	for _, c := range testCases {
		SetDefaults_TfJob(c.in)
		if !reflect.DeepEqual(c.in, c.expected) {
			t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(c.in))
		}
	}
}
