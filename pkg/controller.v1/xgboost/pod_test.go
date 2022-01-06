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
// limitations under the License.

package xgboost

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

	xgboostv1 "github.com/kubeflow/training-operator/pkg/apis/xgboost/v1"
	"github.com/kubeflow/training-operator/pkg/common/util/v1/testutil"
)

func TestClusterSpec(t *testing.T) {
	type tc struct {
		job                 *xgboostv1.XGBoostJob
		rt                  commonv1.ReplicaType
		index               string
		expectedClusterSpec map[string]string
	}
	testCase := []tc{
		tc{
			job:                 testutil.NewXGBoostJobWithMaster(0),
			rt:                  xgboostv1.XGBoostReplicaTypeMaster,
			index:               "0",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "1", "MASTER_PORT": "9999", "RANK": "0", "MASTER_ADDR": "test-xgboostjob-master-0"},
		},
		tc{
			job:                 testutil.NewXGBoostJobWithMaster(1),
			rt:                  xgboostv1.XGBoostReplicaTypeMaster,
			index:               "1",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "2", "MASTER_PORT": "9999", "RANK": "1", "MASTER_ADDR": "test-xgboostjob-master-0", "WORKER_PORT": "9999", "WORKER_ADDRS": "test-xgboostjob-worker-0"},
		},
		tc{
			job:                 testutil.NewXGBoostJobWithMaster(2),
			rt:                  xgboostv1.XGBoostReplicaTypeMaster,
			index:               "0",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "9999", "RANK": "0", "MASTER_ADDR": "test-xgboostjob-master-0", "WORKER_PORT": "9999", "WORKER_ADDRS": "test-xgboostjob-worker-0,test-xgboostjob-worker-1"},
		},
		tc{
			job:                 testutil.NewXGBoostJobWithMaster(2),
			rt:                  xgboostv1.XGBoostReplicaTypeWorker,
			index:               "0",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "9999", "RANK": "1", "MASTER_ADDR": "test-xgboostjob-master-0", "WORKER_PORT": "9999", "WORKER_ADDRS": "test-xgboostjob-worker-0,test-xgboostjob-worker-1"},
		},
		tc{
			job:                 testutil.NewXGBoostJobWithMaster(2),
			rt:                  xgboostv1.XGBoostReplicaTypeWorker,
			index:               "1",
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "9999", "RANK": "2", "MASTER_ADDR": "test-xgboostjob-master-0", "WORKER_PORT": "9999", "WORKER_ADDRS": "test-xgboostjob-worker-0,test-xgboostjob-worker-1"},
		},
	}
	for _, c := range testCase {
		demoTemplateSpec := c.job.Spec.XGBReplicaSpecs[commonv1.ReplicaType(c.rt)].Template
		if err := SetPodEnv(c.job, &demoTemplateSpec, string(c.rt), c.index); err != nil {
			t.Errorf("Failed to set cluster spec: %v", err)
		}
		actual := demoTemplateSpec.Spec.Containers[0].Env
		for _, env := range actual {
			if val, ok := c.expectedClusterSpec[env.Name]; ok {
				if val != env.Value {
					t.Errorf("For name %s Got %s. Expected %s ", env.Name, env.Value, c.expectedClusterSpec[env.Name])
				}
			}
		}
	}
}
