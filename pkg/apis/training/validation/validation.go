// Copyright 2021 The Kubeflow Authors
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

package validation

import (
	"fmt"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	trainingv1 "github.com/kubeflow/training-operator/pkg/apis/training/v1"
	log "github.com/sirupsen/logrus"
)

func ValidateV1MpiJobSpec(c *trainingv1.MPIJobSpec) error {
	if c.MPIReplicaSpecs == nil {
		return fmt.Errorf("MPIReplicaSpecs is not valid")
	}
	launcherExists := false
	for rType, value := range c.MPIReplicaSpecs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("MPIReplicaSpecs is not valid: containers definition expected in %v", rType)
		}
		// Make sure the replica type is valid.
		validReplicaTypes := []commonv1.ReplicaType{trainingv1.MPIReplicaTypeLauncher, trainingv1.MPIReplicaTypeWorker}

		isValidReplicaType := false
		for _, t := range validReplicaTypes {
			if t == rType {
				isValidReplicaType = true
				break
			}
		}
		if !isValidReplicaType {
			return fmt.Errorf("MPIReplicaType is %v but must be one of %v", rType, validReplicaTypes)
		}

		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				msg := fmt.Sprintf("MPIReplicaSpec is not valid: Image is undefined in the container of %v", rType)
				return fmt.Errorf(msg)
			}

			if container.Name == "" {
				msg := fmt.Sprintf("MPIReplicaSpec is not valid: ImageName is undefined in the container of %v", rType)
				return fmt.Errorf(msg)
			}
		}
		if rType == trainingv1.MPIReplicaTypeLauncher {
			launcherExists = true
			if value.Replicas != nil && int(*value.Replicas) != 1 {
				return fmt.Errorf("MPIReplicaSpec is not valid: There must be only 1 launcher replica")
			}
		}

	}

	if !launcherExists {
		return fmt.Errorf("MPIReplicaSpec is not valid: Master ReplicaSpec must be present")
	}
	return nil

}

// ValidateV1MXJobSpec checks that the v1.MXJobSpec is valid.
func ValidateV1MXJobSpec(c *trainingv1.MXJobSpec) error {
	return validateMXJobV1ReplicaSpecs(c.MXReplicaSpecs)
}

// IsScheduler returns true if the type is Scheduler.
func IsScheduler(typ commonv1.ReplicaType) bool {
	return typ == trainingv1.MXReplicaTypeScheduler
}

func validateMXJobV1ReplicaSpecs(specs map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {
	if specs == nil {
		return fmt.Errorf("MXJobSpec is not valid")
	}
	foundScheduler := 0
	for rType, value := range specs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("MXJobSpec is not valid")
		}
		if IsScheduler(rType) {
			foundScheduler++
		}
		// Make sure the image is defined in the container.
		numNamedMXNet := 0
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				log.Warn("Image is undefined in the container")
				return fmt.Errorf("MXJobSpec is not valid")
			}
			if container.Name == trainingv1.MXDefaultContainerName {
				numNamedMXNet++
			}
		}
		// Make sure there has at least one container named "mxnet".
		if numNamedMXNet == 0 {
			log.Warnf("There is no container named mxnet in %v", rType)
			return fmt.Errorf("MXJobSpec is not valid")
		}
	}
	if foundScheduler > 1 {
		return fmt.Errorf("more than 1 scheduler found")
	}
	return nil
}

func ValidateV1PyTorchJobSpec(c *trainingv1.PyTorchJobSpec) error {
	if c.PyTorchReplicaSpecs == nil {
		return fmt.Errorf("PyTorchJobSpec is not valid")
	}
	for rType, value := range c.PyTorchReplicaSpecs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("PyTorchJobSpec is not valid: containers definition expected in %v", rType)
		}
		// Make sure the replica type is valid.
		validReplicaTypes := []commonv1.ReplicaType{trainingv1.PyTorchReplicaTypeMaster, trainingv1.PyTorchReplicaTypeWorker}

		isValidReplicaType := false
		for _, t := range validReplicaTypes {
			if t == rType {
				isValidReplicaType = true
				break
			}
		}

		if !isValidReplicaType {
			return fmt.Errorf("PyTorchReplicaType is %v but must be one of %v", rType, validReplicaTypes)
		}

		//Make sure the image is defined in the container
		defaultContainerPresent := false
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				msg := fmt.Sprintf("PyTorchJobSpec is not valid: Image is undefined in the container of %v", rType)
				return fmt.Errorf(msg)
			}
			if container.Name == trainingv1.PyTorchDefaultContainerName {
				defaultContainerPresent = true
			}
		}
		//Make sure there has at least one container named "pytorch"
		if !defaultContainerPresent {
			msg := fmt.Sprintf("PyTorchJobSpec is not valid: There is no container named %s in %v", trainingv1.PyTorchDefaultContainerName, rType)
			return fmt.Errorf(msg)
		}
		if rType == trainingv1.PyTorchReplicaTypeMaster {
			if value.Replicas != nil && int(*value.Replicas) != 1 {
				return fmt.Errorf("PyTorchJobSpec is not valid: There must be only 1 master replica")
			}
		}

	}

	return nil

}

// ValidateV1TFJobSpec checks that the v1.TFJobSpec is valid.
func ValidateV1TFJobSpec(c *trainingv1.TFJobSpec) error {
	return validateTFV1ReplicaSpecs(c.TFReplicaSpecs)
}

func validateTFV1ReplicaSpecs(specs map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {
	if specs == nil {
		return fmt.Errorf("TFJobSpec is not valid")
	}
	foundChief := 0
	for rType, value := range specs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("TFJobSpec is not valid: containers definition expected in %v", rType)
		}
		if trainingv1.IsChieforMaster(rType) {
			foundChief++
		}
		// Make sure the image is defined in the container.
		numNamedTensorflow := 0
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				msg := fmt.Sprintf("TFJobSpec is not valid: Image is undefined in the container of %v", rType)
				log.Error(msg)
				return fmt.Errorf(msg)
			}
			if container.Name == trainingv1.TFDefaultContainerName {
				numNamedTensorflow++
			}
		}
		// Make sure there has at least one container named "tensorflow".
		if numNamedTensorflow == 0 {
			msg := fmt.Sprintf("TFJobSpec is not valid: There is no container named %s in %v", trainingv1.TFDefaultContainerName, rType)
			log.Error(msg)
			return fmt.Errorf(msg)
		}
	}
	if foundChief > 1 {
		return fmt.Errorf("TFJobSpec is not valid: more than 1 chief/master found")
	}
	return nil
}

func ValidateV1XGBoostJobSpec(c *trainingv1.XGBoostJobSpec) error {
	if c.XGBReplicaSpecs == nil {
		return fmt.Errorf("XGBoostJobSpec is not valid")
	}
	masterExists := false
	for rType, value := range c.XGBReplicaSpecs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("XGBoostJobSpec is not valid: containers definition expected in %v", rType)
		}
		// Make sure the replica type is valid.
		validReplicaTypes := []commonv1.ReplicaType{trainingv1.XGBoostReplicaTypeMaster, trainingv1.XGBoostReplicaTypeWorker}

		isValidReplicaType := false
		for _, t := range validReplicaTypes {
			if t == rType {
				isValidReplicaType = true
				break
			}
		}

		if !isValidReplicaType {
			return fmt.Errorf("XGBoostReplicaType is %v but must be one of %v", rType, validReplicaTypes)
		}

		//Make sure the image is defined in the container
		defaultContainerPresent := false
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				msg := fmt.Sprintf("XGBoostReplicaType is not valid: Image is undefined in the container of %v", rType)
				return fmt.Errorf(msg)
			}
			if container.Name == trainingv1.XGBoostDefaultContainerName {
				defaultContainerPresent = true
			}
		}
		//Make sure there has at least one container named "xgboost"
		if !defaultContainerPresent {
			msg := fmt.Sprintf("XGBoostReplicaType is not valid: There is no container named %s in %v", trainingv1.XGBoostDefaultContainerName, rType)
			return fmt.Errorf(msg)
		}
		if rType == trainingv1.XGBoostReplicaTypeMaster {
			masterExists = true
			if value.Replicas != nil && int(*value.Replicas) != 1 {
				return fmt.Errorf("XGBoostReplicaType is not valid: There must be only 1 master replica")
			}
		}

	}

	if !masterExists {
		return fmt.Errorf("XGBoostReplicaType is not valid: Master ReplicaSpec must be present")
	}
	return nil

}
