package controllers

import (
	"fmt"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	pytorchv1 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func (r *PyTorchJobReconciler) satisfiedExpectations(job *pytorchv1.PyTorchJob) bool {
	satisfied := false
	key, err := common.KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return false
	}

	for rtype := range job.Spec.PyTorchReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := expectation.GenExpectationPodsKey(key, string(rtype))
		satisfied = satisfied || r.Expectations.SatisfiedExpectations(expectationPodsKey)
		// Check the expectations of the services.
		expectationServicesKey := expectation.GenExpectationServicesKey(key, string(rtype))
		satisfied = satisfied || r.Expectations.SatisfiedExpectations(expectationServicesKey)
	}

	return satisfied
}
