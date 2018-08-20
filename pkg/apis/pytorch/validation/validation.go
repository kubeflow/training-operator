// Copyright 2018 The Kubeflow Authors
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

	log "github.com/sirupsen/logrus"

	v1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1alpha2"
)

// ValidateAlphaTwoPyTorchJobSpec checks that the v1alpha2.PyTorchJobSpec is valid.
func ValidateAlphaTwoPyTorchJobSpec(c *v1alpha2.PyTorchJobSpec) error {
	if c.PyTorchReplicaSpecs == nil {
		return fmt.Errorf("PyTorchJobSpec is not valid")
	}
	for rType, value := range c.PyTorchReplicaSpecs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("PyTorchJobSpec is not valid")
		}
		//Make sure the image is defined in the container
		defaultContainerPresent := false
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				log.Warn("Image is undefined in the container")
				return fmt.Errorf("PyTorchJobSpec is not valid")
			}
			if container.Name == v1alpha2.DefaultContainerName {
				defaultContainerPresent = true
			}
		}
		//Make sure there has at least one container named "pytorch"
		if !defaultContainerPresent {
			log.Warnf("There is no container named pytorch in %v", rType)
			return fmt.Errorf("PyTorchJobSpec is not valid")
		}
		if rType == v1alpha2.PyTorchReplicaTypeMaster {
			if value.Replicas != nil && int(*value.Replicas) != 1 {
				log.Warnf("There must be only 1 master replica")
				return fmt.Errorf("PyTorchJobSpec is not valid")
			}
		}

	}
	return nil
}
