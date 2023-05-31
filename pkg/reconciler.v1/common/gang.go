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

package common

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseGangReconciler defines a basic gang reconciler
type BaseGangReconciler struct {
	Enabled bool
}

// GangSchedulingEnabled returns if gang-scheduling is enabled for all jobs
func (r *BaseGangReconciler) GangSchedulingEnabled() bool {
	return r.Enabled
}

// GetPodGroupName returns the name of PodGroup for this job
func (r *BaseGangReconciler) GetPodGroupName(job client.Object) string {
	return job.GetName()
}
