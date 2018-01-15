package validation

import (
	"errors"
	"fmt"

	tfv1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
	"github.com/tensorflow/k8s/pkg/util"
)

// ValidateTfJobSpec checks that the TfJobSpec is valid.
func ValidateTfJobSpec(c *tfv1.TfJobSpec) error {
	if c.TerminationPolicy == nil || c.TerminationPolicy.Chief == nil  {
		return fmt.Errorf("invalid termination policy: %v", c.TerminationPolicy)
	}

	chiefExists := false

	// Check that each replica has a TensorFlow container and a chief.
	for _, r := range c.ReplicaSpecs {
		found := false
		if r.Template == nil && r.TfReplicaType != tfv1.PS {
			return fmt.Errorf("Replica is missing Template; %v", util.Pformat(r))
		}

		if r.TfReplicaType == tfv1.TfReplicaType(c.TerminationPolicy.Chief.ReplicaName) {
			chiefExists = true
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

	if !chiefExists {
		return fmt.Errorf("Missing ReplicaSpec for chief: %v", c.TerminationPolicy.Chief.ReplicaName)
	}

	if c.TensorBoard != nil && c.TensorBoard.LogDir == "" {
			return fmt.Errorf("tbReplicaSpec.LogDir must be specified")
	}
	return nil
}
