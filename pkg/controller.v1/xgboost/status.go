package xgboost

import (
	"fmt"

	"github.com/sirupsen/logrus"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	commonutil "github.com/kubeflow/training-operator/pkg/util"
)

func setRunningCondition(logger *logrus.Entry, jobName string, jobStatus *kubeflowv1.JobStatus) error {
	msg := fmt.Sprintf("XGBoostJob %s is running.", jobName)
	if condition := findStatusCondition(jobStatus.Conditions, kubeflowv1.JobRunning); condition == nil {
		err := commonutil.UpdateJobConditions(jobStatus, kubeflowv1.JobRunning, xgboostJobRunningReason, msg)
		if err != nil {
			logger.Infof("Append job condition error: %v", err)
			return err
		}
	}
	return nil
}

func findStatusCondition(conditions []kubeflowv1.JobCondition, conditionType kubeflowv1.JobConditionType) *kubeflowv1.JobCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
