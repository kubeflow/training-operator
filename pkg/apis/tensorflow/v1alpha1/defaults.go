package v1alpha1

import (
	"github.com/golang/protobuf/proto"
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_TFJob sets any unspecified values to defaults
func SetDefaults_TFJob(obj *TFJob) {
	c := &obj.Spec

	if c.TFImage == "" {
		c.TFImage = DefaultTFImage
	}

	// Check that each replica has a TensorFlow container.
	for _, r := range c.ReplicaSpecs {

		if r.TFPort == nil {
			r.TFPort = proto.Int32(TFPort)
		}

		if string(r.TFReplicaType) == "" {
			r.TFReplicaType = MASTER
		}

		if r.Replicas == nil {
			r.Replicas = proto.Int32(Replicas)
		}
	}
	if c.TerminationPolicy == nil {
		c.TerminationPolicy = &TerminationPolicySpec{
			Chief: &ChiefSpec{
				ReplicaName:  "MASTER",
				ReplicaIndex: 0,
			},
		}
	}

}
