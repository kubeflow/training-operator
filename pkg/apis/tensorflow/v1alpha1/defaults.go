package v1alpha1

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/tensorflow/k8s/pkg/util"
	"k8s.io/client-go/pkg/api/v1"
)

// SetDefaults sets any unspecified values to defaults
func SetDefaults_TfJobSpec(c *TfJobSpec) error {
	if c.TfImage == "" {
		c.TfImage = DefaultTFImage
	}

	// Check that each replica has a TensorFlow container.
	for _, r := range c.ReplicaSpecs {
		if r.Template == nil && r.TfReplicaType != PS {
			return fmt.Errorf("ReplicaType: %v, Replica is missing Template; %v", r.TfReplicaType, util.Pformat(r))
		}

		if r.TfPort == nil {
			r.TfPort = proto.Int32(TfPort)
		}

		if string(r.TfReplicaType) == "" {
			r.TfReplicaType = MASTER
		}

		if r.Replicas == nil {
			r.Replicas = proto.Int32(Replicas)
		}

		//Set the default configuration for a PS server if the user didn't specify a PodTemplateSpec
		if r.Template == nil && r.TfReplicaType == PS {
			setDefaultPSPodTemplateSpec(r, c.TfImage)
		}
	}
	return nil
}

func setDefaultPSPodTemplateSpec(r *TfReplicaSpec, tfImage string) {
	r.IsDefaultPS = true
	r.Template = &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Image: tfImage,
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
	}
}
