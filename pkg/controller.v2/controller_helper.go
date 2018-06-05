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

package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

const (
	labelGroupName = "group_name"
	labelTFJobKey  = "tf_job_key"

	failedMarshalTFJobReason = "FailedMarshalTFJob"
)

var (
	errPortNotFound = fmt.Errorf("Failed to found the port")
)

func genOwnerReference(tfjob *tfv1alpha2.TFJob) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         tfv1alpha2.SchemeGroupVersion.String(),
		Kind:               tfv1alpha2.Kind,
		Name:               tfjob.Name,
		UID:                tfjob.UID,
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func genLabels(tfjobKey string) map[string]string {
	return map[string]string{
		labelGroupName: tfv1alpha2.GroupName,
		labelTFJobKey:  strings.Replace(tfjobKey, "/", "-", -1),
	}
}

func genGeneralName(tfjobKey, rtype, index string) string {
	n := tfjobKey + "-" + rtype + "-" + index
	return strings.Replace(n, "/", "-", -1)
}

func genDNSRecord(tfjobKey, rtype, index, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", genGeneralName(tfjobKey, rtype, index), namespace)
}

// convertTFJobToUnstructured uses JSON to convert TFJob to Unstructured.
func convertTFJobToUnstructured(tfJob *tfv1alpha2.TFJob) (*unstructured.Unstructured, error) {
	var unstructured unstructured.Unstructured
	b, err := json.Marshal(tfJob)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &unstructured); err != nil {
		return nil, err
	}
	return &unstructured, nil
}

// getPortFromTFJob gets the port of tensorflow container.
func getPortFromTFJob(tfJob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType) (int32, error) {
	containers := tfJob.Spec.TFReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == tfv1alpha2.DefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == tfv1alpha2.DefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, errPortNotFound
}
