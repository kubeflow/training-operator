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
// limitations under the License

package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClearGeneratedFields will clear the generated fields from the given object meta.
// It is used to avoid problems like "the object has been modified; please apply your
// changes to the latest version and try again".
func ClearGeneratedFields(objmeta *metav1.ObjectMeta) {
	objmeta.UID = ""
	objmeta.CreationTimestamp = metav1.Time{}
}
