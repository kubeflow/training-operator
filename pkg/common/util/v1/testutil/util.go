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

package testutil

import (
	"strings"
	"testing"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

const (
	LabelGroupName = "group-name"
	JobNameLabel   = "job-name"
	// Deprecated label. Has to be removed later
	DeprecatedLabelTFJobName = "tf-job-name"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc        = cache.DeletionHandlingMetaNamespaceKeyFunc
	GroupName      = tfv1.GroupName
	ControllerName = "tf-operator"
)

func GenLabels(jobName string) map[string]string {
	return map[string]string{
		LabelGroupName:           GroupName,
		JobNameLabel:             strings.Replace(jobName, "/", "-", -1),
		DeprecatedLabelTFJobName: strings.Replace(jobName, "/", "-", -1),
	}
}

func GenOwnerReference(tfjob *tfv1.TFJob) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         tfv1.SchemeGroupVersion.String(),
		Kind:               tfv1.Kind,
		Name:               tfjob.Name,
		UID:                tfjob.UID,
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

// ConvertTFJobToUnstructured uses function ToUnstructured to convert TFJob to Unstructured.
func ConvertTFJobToUnstructured(tfJob *tfv1.TFJob) (*unstructured.Unstructured, error) {
	object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tfJob)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{
		Object: object,
	}, nil
}

func GetKey(tfJob *tfv1.TFJob, t *testing.T) string {
	key, err := KeyFunc(tfJob)
	if err != nil {
		t.Errorf("Unexpected error getting key for job %v: %v", tfJob.Name, err)
		return ""
	}
	return key
}

func CheckCondition(tfJob *tfv1.TFJob, condition common.JobConditionType, reason string) bool {
	for _, v := range tfJob.Status.Conditions {
		if v.Type == condition && v.Status == v1.ConditionTrue && v.Reason == reason {
			return true
		}
	}
	return false
}
