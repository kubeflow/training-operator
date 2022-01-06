// Copyright 2022 The Kubeflow Authors
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

package mxnet

import (
	"context"

	mxjobv1 "github.com/kubeflow/training-operator/pkg/apis/mxnet/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubeflow/training-operator/pkg/common/util/v1/testutil"
)

var _ = Describe("MXJob controller", func() {
	Context("Test Add MXJob", func() {
		It("should get desired MXJob", func() {
			mxJob := testutil.NewMXJob(1, 0)

			ctx := context.Background()
			Expect(testK8sClient.Create(ctx, mxJob)).Should(Succeed())

			obj := &mxjobv1.MXJob{}
			key := types.NamespacedName{
				Namespace: mxJob.GetNamespace(),
				Name:      mxJob.GetName(),
			}
			Eventually(func() error {
				return testK8sClient.Get(ctx, key, obj)
			}).Should(BeNil())
		})
	})

})
