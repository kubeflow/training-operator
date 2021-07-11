package validation

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	mxv1 "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestValidateV1MXJobSpec(t *testing.T) {
	testCases := []mxv1.MXJobSpec{
		{
			MXReplicaSpecs: nil,
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				mxv1.MXReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				mxv1.MXReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "",
								},
							},
						},
					},
				},
			},
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				mxv1.MXReplicaTypeWorker: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Name:  "",
									Image: "mxjob/mxnet:gpu",
								},
							},
						},
					},
				},
			},
		},
		{
			MXReplicaSpecs: map[commonv1.ReplicaType]*commonv1.ReplicaSpec{
				mxv1.MXReplicaTypeScheduler: &commonv1.ReplicaSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
	}
	for _, c := range testCases {
		err := ValidateV1MXJobSpec(&c)
		if err.Error() != "MXJobSpec is not valid" {
			t.Error("Failed validate the alpha2.MXJobSpec")
		}
	}
}
