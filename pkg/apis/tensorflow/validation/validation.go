package validation

import (
	"errors"
	"fmt"

	tfv1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	"github.com/tensorflow/k8s/pkg/util"
)

// ValidateTFJobSpec checks that the TFJobSpec is valid.
func ValidateTFJobSpec(c *tfv1.TFJobSpec) error {
	if c.TerminationPolicy == nil || c.TerminationPolicy.Chief == nil {
		return fmt.Errorf("invalid termination policy: %v", c.TerminationPolicy)
	}

	chiefExists := false

	// Check that each replica has a TensorFlow container and a chief.
	for _, r := range c.ReplicaSpecs {
		found := false
		if r.Template == nil {
			return fmt.Errorf("Replica is missing Template; %v", util.Pformat(r))
		}

		if r.TFReplicaType == tfv1.TFReplicaType(c.TerminationPolicy.Chief.ReplicaName) {
			chiefExists = true
		}

		if r.TFPort == nil {
			return errors.New("tfReplicaSpec.TFPort can't be nil.")
		}

		// Make sure the replica type is valid.
		validReplicaTypes := []tfv1.TFReplicaType{tfv1.MASTER, tfv1.PS, tfv1.WORKER}

		isValidReplicaType := false
		for _, t := range validReplicaTypes {
			if t == r.TFReplicaType {
				isValidReplicaType = true
				break
			}
		}

		if !isValidReplicaType {
			return fmt.Errorf("tfReplicaSpec.TFReplicaType is %v but must be one of %v", r.TFReplicaType, validReplicaTypes)
		}

		for _, c := range r.Template.Spec.Containers {
			if c.Name == string(tfv1.TENSORFLOW) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Replica type %v is missing a container named %v", r.TFReplicaType, tfv1.TENSORFLOW)
		}
	}

	if !chiefExists {
		return fmt.Errorf("Missing ReplicaSpec for chief: %v", c.TerminationPolicy.Chief.ReplicaName)
	}

	if c.TensorBoard != nil && c.TensorBoard.LogDir == "" {
		return fmt.Errorf("tbReplicaSpec.LogDir must be specified")
	}
	return nil
}
