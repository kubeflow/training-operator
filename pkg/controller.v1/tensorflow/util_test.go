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

package tensorflow

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func TestGenOwnerReference(t *testing.T) {
	testName := "test-tfjob"
	testUID := uuid.NewUUID()
	tfJob := &kubeflowv1.TFJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
			UID:  testUID,
		},
	}

	ref := reconciler.GenOwnerReference(tfJob)
	if ref.UID != testUID {
		t.Errorf("Expected UID %s, got %s", testUID, ref.UID)
	}
	if ref.Name != testName {
		t.Errorf("Expected Name %s, got %s", testName, ref.Name)
	}
	if ref.APIVersion != kubeflowv1.SchemeGroupVersion.String() {
		t.Errorf("Expected APIVersion %s, got %s", kubeflowv1.SchemeGroupVersion.String(), ref.APIVersion)
	}
}

func TestGenLabels(t *testing.T) {
	testJobName := "test/key"
	expctedVal := "test-key"

	labels := reconciler.GenLabels(testJobName)
	jobNameLabel := commonv1.JobNameLabel
	JobNameLabelDeprecated := commonv1.JobNameLabelDeprecated

	if labels[jobNameLabel] != expctedVal {
		t.Errorf("Expected %s %s, got %s", jobNameLabel, expctedVal, jobNameLabel)
	}

	if labels[JobNameLabelDeprecated] != expctedVal {
		t.Errorf("Expected %s %s, got %s", JobNameLabelDeprecated, expctedVal, JobNameLabelDeprecated)
	}

	if labels[commonv1.GroupNameLabelDeprecated] != kubeflowv1.GroupVersion.Group {
		t.Errorf("Expected %s %s, got %s", commonv1.GroupNameLabelDeprecated, kubeflowv1.GroupVersion.Group,
			labels[commonv1.GroupNameLabelDeprecated])
	}

	if labels[commonv1.OperatorNameLabel] != controllerName {
		t.Errorf("Expected %s %s, got %s", commonv1.OperatorNameLabel, controllerName,
			labels[commonv1.OperatorNameLabel])
	}
}
