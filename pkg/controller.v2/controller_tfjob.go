package controller

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes/scheme"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

// When a pod is added, set the defaults and enqueue the current tfjob.
func (tc *TFJobController) addTFJob(obj interface{}) {
	// Convert from unstructured object.
	tfjob, err := tfJobFromUnstructured(obj)
	if err != nil {
		log.Errorf("Failed to convert the TFJob: %v", err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to TFJob object: %v", err)
			log.Warn(errMsg)
			// TODO(gaocegege): Set the condition or publish an event to tell the users.
		}
		return
	}

	// Set default for the new tfjob.
	scheme.Scheme.Default(tfjob)

	msg := fmt.Sprintf("TFJob %s is created.", tfjob.Name)
	log.Info(msg)

	// Add a created condition.
	err = tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobCreated, tfJobCreatedReason, msg)
	if err != nil {
		log.Infof("Append tfjob condition error: %v", err)
		return
	}

	tc.enqueueTFJob(obj)
}

// When a pod is updated, enqueue the current tfjob.
func (tc *TFJobController) updateTFJob(old, cur interface{}) {
	oldTFJob, err := tfJobFromUnstructured(old)
	if err != nil {
		return
	}
	log.Infof("Updating tfjob: %s", oldTFJob.Name)
	tc.enqueueTFJob(cur)
}
