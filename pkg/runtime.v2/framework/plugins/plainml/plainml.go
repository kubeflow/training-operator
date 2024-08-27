/*
Copyright 2024 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plainml

import (
	"context"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtime "github.com/kubeflow/training-operator/pkg/runtime.v2"
	"github.com/kubeflow/training-operator/pkg/runtime.v2/framework"
)

var _ framework.EnforceMLPolicyPlugin = (*PlainML)(nil)

type PlainML struct{}

const Name = "PlainML"

func New(context.Context, client.Client, client.FieldIndexer) (framework.Plugin, error) {
	return &PlainML{}, nil
}

func (p *PlainML) Name() string {
	return Name
}

func (p *PlainML) EnforceMLPolicy(info *runtime.Info) error {
	if info == nil || info.MLPolicy == nil || info.MLPolicy.Torch != nil || info.MLPolicy.MPI != nil {
		return nil
	}
	numNodes := ptr.Deref(info.MLPolicy.NumNodes, 1)
	for rName := range info.TotalRequests {
		info.TotalRequests[rName] = runtime.TotalResourceRequest{
			Replicas:    numNodes,
			PodRequests: info.TotalRequests[rName].PodRequests,
		}
	}
	return nil
}
