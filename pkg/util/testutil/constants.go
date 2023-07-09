package testutil

import (
	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
)

const (
	Timeout            = 30 * time.Second
	Interval           = 250 * time.Millisecond
	ConsistentDuration = 3 * time.Second
)

var (
	IgnoreJobConditionsTimes = cmpopts.IgnoreFields(kubeflowv1.JobCondition{}, "LastUpdateTime", "LastTransitionTime")
)
