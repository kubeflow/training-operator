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
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

func TestGenOwnerReference(t *testing.T) {
	testName := "test-tfjob"
	testUID := types.UID("test-UID")
	tfJob := &tfv1alpha2.TFJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
			UID:  testUID,
		},
	}

	ref := genOwnerReference(tfJob)
	if ref.UID != testUID {
		t.Errorf("Expected UID %s, got %s", testUID, ref.UID)
	}
	if ref.Name != testName {
		t.Errorf("Expected Name %s, got %s", testName, ref.Name)
	}
	if ref.APIVersion != tfv1alpha2.SchemeGroupVersion.String() {
		t.Errorf("Expected APIVersion %s, got %s", tfv1alpha2.SchemeGroupVersion.String(), ref.APIVersion)
	}
}

func TestGenLabels(t *testing.T) {
	testKey := "test/key"
	expctedKey := "test-key"

	labels := genLabels(testKey)

	if labels[labelTFJobKey] != expctedKey {
		t.Errorf("Expected %s %s, got %s", labelTFJobKey, expctedKey, labels[labelTFJobKey])
	}
	if labels[labelGroupName] != tfv1alpha2.GroupName {
		t.Errorf("Expected %s %s, got %s", labelGroupName, tfv1alpha2.GroupName, labels[labelGroupName])
	}
}

func TestGenGeneralName(t *testing.T) {
	testRType := "worker"
	testIndex := "1"
	testKey := "1/2/3/4/5"
	expectedName := fmt.Sprintf("1-2-3-4-5-%s-%s", testRType, testIndex)

	name := genGeneralName(testKey, testRType, testIndex)
	if name != expectedName {
		t.Errorf("Expected name %s, got %s", expectedName, name)
	}
}

func TestConvertTFJobToUnstructured(t *testing.T) {
	testName := "test-tfjob"
	testUID := types.UID("test-UID")
	tfJob := &tfv1alpha2.TFJob{
		TypeMeta: metav1.TypeMeta{
			Kind: tfv1alpha2.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
			UID:  testUID,
		},
	}

	_, err := convertTFJobToUnstructured(tfJob)
	if err != nil {
		t.Errorf("Expected error to be nil while got %v", err)
	}
}
