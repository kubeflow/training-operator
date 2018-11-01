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

package jobcontroller

import (
	"fmt"
	"testing"
)

func TestGenGeneralName(t *testing.T) {
	testRType := "worker"
	testIndex := "1"
	testKey := "1/2/3/4/5"
	expectedName := fmt.Sprintf("1-2-3-4-5-%s-%s", testRType, testIndex)

	name := GenGeneralName(testKey, testRType, testIndex)
	if name != expectedName {
		t.Errorf("Expected name %s, got %s", expectedName, name)
	}
}
