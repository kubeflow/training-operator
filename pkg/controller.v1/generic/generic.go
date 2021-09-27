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

package generic

import (
	"context"
	"fmt"

	genericv1 "github.com/kubeflow/tf-operator/pkg/apis/generic/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ErrFailedConvertToGenericJob = "failed to convert client.Object to *GenericJob"

// GetReconcilerName returns the name of this reconciler, which is "Kubeflow Reconciler"
func (r *KubeflowReconciler) GetReconcilerName() string {
	return "generic-job-reconciler"
}

func (r *KubeflowReconciler) GetJob(ctx context.Context, req ctrl.Request) (client.Object, error) {
	job := &genericv1.GenericJob{}
	err := r.Get(ctx, req.NamespacedName, job)
	return job, err
}

func (r *KubeflowReconciler) ExtractReplicasSpec(job client.Object) (map[commonv1.ReplicaType]*commonv1.ReplicaSpec, error) {
	realJob, ok := job.(*genericv1.GenericJob)
	if !ok {
		return nil, fmt.Errorf(ErrFailedConvertToGenericJob)
	}
	return realJob.Spec.ReplicaSpecs, nil
}

func (r *KubeflowReconciler) ExtractRunPolicy(job client.Object) (*commonv1.RunPolicy, error) {
	realJob, ok := job.(*genericv1.GenericJob)
	if !ok {
		return nil, fmt.Errorf(ErrFailedConvertToGenericJob)
	}
	return &realJob.Spec.RunPolicy, nil
}

func (r *KubeflowReconciler) ExtractJobStatus(job client.Object) (*commonv1.JobStatus, error) {
	realJob, ok := job.(*genericv1.GenericJob)
	if !ok {
		return nil, fmt.Errorf(ErrFailedConvertToGenericJob)
	}
	return &realJob.Status, nil
}

func (r *KubeflowReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool {
	return false
}
