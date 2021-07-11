/*
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

package xgboost

import (
	"fmt"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	v1 "github.com/kubeflow/tf-operator/pkg/apis/xgboost/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// satisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (r *XGBoostJobReconciler) satisfiedExpectations(xgbJob *v1.XGBoostJob) bool {
	satisfied := false
	key, err := common.KeyFunc(xgbJob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", xgbJob, err))
		return false
	}
	for rtype := range xgbJob.Spec.XGBReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := expectation.GenExpectationPodsKey(key, string(rtype))
		satisfied = satisfied || r.Expectations.SatisfiedExpectations(expectationPodsKey)
		// Check the expectations of the services.
		expectationServicesKey := expectation.GenExpectationServicesKey(key, string(rtype))
		satisfied = satisfied || r.Expectations.SatisfiedExpectations(expectationServicesKey)
	}
	return satisfied
}
