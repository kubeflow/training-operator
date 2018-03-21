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

package v1alpha2

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Pformat returns a pretty format output of any value that can be marshalled to JSON.
func Pformat(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	valueJSON, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Warningf("Couldn't pretty format %v, error: %v", value, err)
		return fmt.Sprintf("%v", value)
	}
	return string(valueJSON)
}
