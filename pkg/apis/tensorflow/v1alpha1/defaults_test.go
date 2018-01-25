package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/tensorflow/k8s/pkg/util"
	"k8s.io/api/core/v1"
)

func TestSetDefaults_TFJob(t *testing.T) {
	type testCase struct {
		in       *TFJob
		expected *TFJob
	}

	testCases := []testCase{
		{
			in: &TFJob{
				Spec: TFJobSpec{
					ReplicaSpecs: []*TFReplicaSpec{
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
					TFImage: "tensorflow/tensorflow:1.3.0",
				},
			},
			expected: &TFJob{
				Spec: TFJobSpec{
					ReplicaSpecs: []*TFReplicaSpec{
						{
							Replicas: proto.Int32(1),
							TFPort:   proto.Int32(2222),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "tensorflow",
										},
									},
								},
							},
							TFReplicaType: MASTER,
						},
					},
					TFImage: "tensorflow/tensorflow:1.3.0",
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
			in: &TFJob{
				Spec: TFJobSpec{
					ReplicaSpecs: []*TFReplicaSpec{
						{
							TFReplicaType: PS,
						},
					},
					TFImage: "tensorflow/tensorflow:1.3.0",
				},
			},
			expected: &TFJob{
				Spec: TFJobSpec{
					ReplicaSpecs: []*TFReplicaSpec{
						{
							Replicas:      proto.Int32(1),
							TFPort:        proto.Int32(2222),
							TFReplicaType: PS,
						},
					},
					TFImage: "tensorflow/tensorflow:1.3.0",
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
		SetDefaults_TFJob(c.in)
		if !reflect.DeepEqual(c.in, c.expected) {
			t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(c.in))
		}
	}
}
