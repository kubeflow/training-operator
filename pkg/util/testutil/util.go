package testutil

import (
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

func GetKey(tfJob *tfv1alpha2.TFJob, t *testing.T) string {
	key, err := KeyFunc(tfJob)
	if err != nil {
		t.Errorf("Unexpected error getting key for job %v: %v", tfJob.Name, err)
		return ""
	}
	return key
}

func CheckCondition(tfJob *tfv1alpha2.TFJob, condition tfv1alpha2.TFJobConditionType, reason string) bool {
	for _, v := range tfJob.Status.Conditions {
		if v.Type == condition && v.Status == v1.ConditionTrue && v.Reason == reason {
			return true
		}
	}
	return false
}
