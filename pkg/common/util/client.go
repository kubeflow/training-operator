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

package util

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// TODO (Jeffwan@): Find an elegant way to either use delegatingReader or directly use clientss
// GetDelegatingClientFromClient try to extract client reader from client, client
// reader reads cluster info from api client.
func GetDelegatingClientFromClient(c client.Client) (client.Client, error) {
	input := client.NewDelegatingClientInput{
		CacheReader:     nil,
		Client:          c,
		UncachedObjects: nil,
	}
	return client.NewDelegatingClient(input)
}
