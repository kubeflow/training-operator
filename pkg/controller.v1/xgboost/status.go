package xgboost

import (
	"fmt"

	"github.com/sirupsen/logrus"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
)

func setRunningCondition(logger *logrus.Entry, jobName string, jobStatus *commonv1.JobStatus) error {
	msg := fmt.Sprintf("XGBoostJob %s is running.", jobName)
	if condition := findStatusCondition(jobStatus.Conditions, commonv1.JobRunning); condition == nil {
		err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, xgboostJobRunningReason, msg)
		if err != nil {
			logger.Infof("Append job condition error: %v", err)
			return err
		}
	}
	return nil
}

func findStatusCondition(conditions []commonv1.JobCondition, conditionType commonv1.JobConditionType) *commonv1.JobCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
