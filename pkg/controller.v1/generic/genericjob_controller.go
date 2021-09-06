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

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/kubeflow/common/pkg/reconciler.v1/common"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type GenericJobReconciler struct {
	common.KubeflowReconciler
	client.Client
}

func NewReonciler(mgr manager.Manager, enableGangScheduling bool) *GenericJobReconciler {
	baseKubeflowReconciler := common.BareKubeflowReconciler()
	// Generate Bare Components
	jobInter := common.BareKubeflowJobReconciler(mgr.GetClient())
	podInter := common.BareKubeflowPodReconciler(mgr.GetClient())
	svcInter := common.BareKubeflowServiceReconciler(mgr.GetClient())
	gangInter := common.BareVolcanoReconciler(mgr.GetClient(), nil, enableGangScheduling)
	utilInter := common.BareUtilReconciler(nil, logr.FromContext(context.Background()), mgr.GetScheme())

	// Assign interfaces for jobInterface
	jobInter.PodInterface = podInter
	jobInter.ServiceInterface = svcInter
	jobInter.GangSchedulingInterface = gangInter
	jobInter.ReconcilerUtilInterface = utilInter

	// Assign interfaces for podInterface
	podInter.JobInterface = jobInter
	podInter.GangSchedulingInterface = gangInter
	podInter.ReconcilerUtilInterface = utilInter

	// Assign interfaces for svcInterface
	svcInter.PodInterface = podInter
	svcInter.JobInterface = jobInter
	svcInter.ReconcilerUtilInterface = utilInter

	// Assign interfaces for gangInterface
	gangInter.ReconcilerUtilInterface = utilInter

	// Prepare KubeflowReconciler
	baseKubeflowReconciler.JobInterface = jobInter
	baseKubeflowReconciler.PodInterface = podInter
	baseKubeflowReconciler.ServiceInterface = svcInter
	baseKubeflowReconciler.GangSchedulingInterface = gangInter
	baseKubeflowReconciler.ReconcilerUtilInterface = utilInter

	reconciler := &GenericJobReconciler{
		KubeflowReconciler: *baseKubeflowReconciler,
		Client:             mgr.GetClient(),
	}
	reconciler.OverrideForKubeflowReconcilerInterface(reconciler, reconciler, reconciler, reconciler, reconciler)

	return reconciler
}
