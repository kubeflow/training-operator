package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/tensorflow/k8s/pkg/util"
	"k8s.io/api/core/v1"
)

func TestSetDefaults_TfJobSpec(t *testing.T) {
	type testCase struct {
		in       *TfJobSpec
		expected *TfJobSpec
	}

	testCases := []testCase{
		{
			in: &TfJobSpec{
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
			expected: &TfJobSpec{
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
		{
			in: &TfJobSpec{
				ReplicaSpecs: []*TfReplicaSpec{
					{
						TfReplicaType: PS,
					},
				},
				TfImage: "tensorflow/tensorflow:1.3.0",
			},
			expected: &TfJobSpec{
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
	}

	for _, c := range testCases {
		if err := SetDefaults_TfJobSpec(c.in); err != nil {
			t.Errorf("SetDefaults error; %v", err)
		}
		if !reflect.DeepEqual(c.in, c.expected) {
			t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(c.in))
		}
	}
}
