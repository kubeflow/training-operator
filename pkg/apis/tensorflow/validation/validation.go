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

	common "github.com/kubeflow/tf-operator/pkg/apis/common/v1beta1"
	tfv2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tfv1beta1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1beta1"
)

// ValidateBetaOneTFJobSpec checks that the v1beta1.TFJobSpec is valid.
func ValidateBetaOneTFJobSpec(c *tfv1beta1.TFJobSpec) error {
	return validateBetaOneReplicaSpecs(c.TFReplicaSpecs)
}

func validateBetaOneReplicaSpecs(specs map[tfv1beta1.TFReplicaType]*common.ReplicaSpec) error {
	if specs == nil {
		return fmt.Errorf("TFJobSpec is not valid")
	}
	foundChief := 0
	var foundEvaluator int32 = 0
	for rType, value := range specs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("TFJobSpec is not valid")
		}
		if tfv1beta1.IsChieforMaster(rType) {
			foundChief++
		}
		if tfv1beta1.IsEvaluator(rType) {
			foundEvaluator = foundEvaluator + *value.Replicas
		}
		// Make sure the image is defined in the container.
		numNamedTensorflow := 0
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				log.Warn("Image is undefined in the container")
				return fmt.Errorf("TFJobSpec is not valid")
			}
			if container.Name == tfv1beta1.DefaultContainerName {
				numNamedTensorflow++
			}
		}
		// Make sure there has at least one container named "tensorflow".
		if numNamedTensorflow == 0 {
			log.Warnf("There is no container named tensorflow in %v", rType)
			return fmt.Errorf("TFJobSpec is not valid")
		}
	}
	if foundChief > 1 {
		return fmt.Errorf("more than 1 chief/master found")
	}
	if foundEvaluator > 1 {
		return fmt.Errorf("more than 1 evaluator found")
	}
	return nil
}

// ValidateAlphaTwoTFJobSpec checks that the v1alpha2.TFJobSpec is valid.
func ValidateAlphaTwoTFJobSpec(c *tfv2.TFJobSpec) error {
	return validateAlphaTwoReplicaSpecs(c.TFReplicaSpecs)
}

func validateAlphaTwoReplicaSpecs(specs map[tfv2.TFReplicaType]*tfv2.TFReplicaSpec) error {
	if specs == nil {
		return fmt.Errorf("TFJobSpec is not valid")
	}
	foundChief := 0
	var foundEvaluator int32 = 0
	for rType, value := range specs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("TFJobSpec is not valid")
		}
		if tfv2.IsChieforMaster(rType) {
			foundChief++
		}
		if tfv2.IsEvaluator(rType) {
			foundEvaluator = foundEvaluator + *value.Replicas
		}
		// Make sure the image is defined in the container.
		numNamedTensorflow := 0
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				log.Warn("Image is undefined in the container")
				return fmt.Errorf("TFJobSpec is not valid")
			}
			if container.Name == tfv2.DefaultContainerName {
				numNamedTensorflow++
			}
		}
		// Make sure there has at least one container named "tensorflow".
		if numNamedTensorflow == 0 {
			log.Warnf("There is no container named tensorflow in %v", rType)
			return fmt.Errorf("TFJobSpec is not valid")
		}
	}
	if foundChief > 1 {
		return fmt.Errorf("more than 1 chief/master found")
	}
	if foundEvaluator > 1 {
		return fmt.Errorf("more than 1 evaluator found")
	}
	return nil
}
