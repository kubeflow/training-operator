package controller

import (
	log "github.com/sirupsen/logrus"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

func loggerForTFJob(tfjob *tfv1alpha2.TFJob) *log.Entry {
	return log.WithFields(log.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job": tfjob.ObjectMeta.Namespace + "/" + tfjob.ObjectMeta.Name,
		"uid": tfjob.ObjectMeta.UID,
	})
}
