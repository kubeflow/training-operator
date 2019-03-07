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

package kbutil

import (
	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
)

var kubeBatchClientSet kubebatchclient.Interface

//SetKubeBatchClientSet sets kube-batch client set.
func SetKubeBatchClientInterface(clientset kubebatchclient.Interface) {
	kubeBatchClientSet = clientset
}

// GetKubeBatchClientSet gets kube-batch client set.
func GetKubeBatchClientInterface() kubebatchclient.Interface {
	return kubeBatchClientSet
}
