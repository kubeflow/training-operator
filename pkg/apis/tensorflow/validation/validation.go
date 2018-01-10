package validation

import (
	"errors"
	"fmt"

	tfv1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	"github.com/tensorflow/k8s/pkg/util"
)

// ValidateTfJobSpec checks that the TfJobSpec is valid.
func ValidateTfJobSpec(c *tfv1.TfJobSpec) error {
	// Check that each replica has a TensorFlow container.
	for _, r := range c.ReplicaSpecs {
		found := false
		if r.Template == nil && r.TfReplicaType != tfv1.PS {
			return fmt.Errorf("Replica is missing Template; %v", util.Pformat(r))
		}

		if r.TfReplicaType == tfv1.MASTER && *r.Replicas != 1 {
			return errors.New("The MASTER must have Replicas = 1")
		}

		if r.TfPort == nil {
			return errors.New("tfReplicaSpec.TfPort can't be nil.")
		}

		// Make sure the replica type is valid.
		validReplicaTypes := []tfv1.TfReplicaType{tfv1.MASTER, tfv1.PS, tfv1.WORKER}

		isValidReplicaType := false
		for _, t := range validReplicaTypes {
			if t == r.TfReplicaType {
				isValidReplicaType = true
				break
			}
		}

		if !isValidReplicaType {
			return fmt.Errorf("tfReplicaSpec.TfReplicaType is %v but must be one of %v", r.TfReplicaType, validReplicaTypes)
		}

		for _, c := range r.Template.Spec.Containers {
			if c.Name == string(tfv1.TENSORFLOW) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Replica type %v is missing a container named %v", r.TfReplicaType, tfv1.TENSORFLOW)
		}
	}

	if c.TerminationPolicy != nil {
		if c.TerminationPolicy.Chief == nil {
			return errors.New("invalid termination policy, Chief cannot be nil")
		}
		if c.TerminationPolicy.Chief.ReplicaName != "MASTER" || c.TerminationPolicy.Chief.ReplicaIndex != 0 {
			return errors.New("invalid termination policy, Chief should have replicaName=MASTER and index=0")
		}
	}

	return nil
}
