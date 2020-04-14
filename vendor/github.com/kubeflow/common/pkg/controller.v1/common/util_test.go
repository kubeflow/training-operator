// Copyright 2019 The Kubeflow Authors
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

package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
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

func TestMaxInt(t *testing.T) {
	type testCase struct {
		x           int
		y           int
		expectedMax int
	}
	var testCases = []testCase{
		{
			x:           10,
			y:           20,
			expectedMax: 20,
		},
		{
			x:           20,
			y:           10,
			expectedMax: 20,
		},
		{
			x:           5,
			y:           5,
			expectedMax: 5,
		},
	}

	for _, tc := range testCases {
		result := MaxInt(tc.x, tc.y)
		assert.Equal(t, tc.expectedMax, result)
	}
}
