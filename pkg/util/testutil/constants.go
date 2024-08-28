package testutil

import (
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

const (
	Timeout            = 30 * time.Second
	Interval           = 250 * time.Millisecond
	ConsistentDuration = 3 * time.Second
)

var (
	IgnoreJobConditionsTimes = cmpopts.IgnoreFields(kubeflowv1.JobCondition{}, "LastUpdateTime", "LastTransitionTime")
	MalformedManagedBy       = "other-job-controller"
)
