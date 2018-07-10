package controller

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

const (
	failedMarshalTFJobReason  = "FailedMarshalTFJob"
	terminatedTFJobReason     = "TFJobTerminated"
	FailedDeletePdbReason     = "FailedDeletePdb"
	SuccessfulDeletePdbReason = "SuccessfulDeletePdb"
)
// When a pod is added, set the defaults and enqueue the current tfjob.
func (tc *TFJobController) addTFJob(obj interface{}) {
	// Convert from unstructured object.
	tfJob, err := tfJobFromUnstructured(obj)
	if err != nil {
		log.Errorf("Failed to convert the TFJob: %v", err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to TFJob object: %v", err)
			log.Warn(errMsg)
			tc.recorder.Event(tfJob, v1.EventTypeWarning, failedMarshalTFJobReason, errMsg)
		}
		return
	}

	// Set default for the new tfjob.
	scheme.Scheme.Default(tfJob)

	msg := fmt.Sprintf("TFJob %s is created.", tfJob.Name)
	log.Info(msg)

	// Add a created condition.
	err = updateTFJobConditions(tfJob, tfv1alpha2.TFJobCreated, tfJobCreatedReason, msg)
	if err != nil {
		log.Infof("Append tfJob condition error: %v", err)
		return
	}

	// Convert from tfjob object
	err = unstructuredFromTFJob(obj, tfJob)
	if err != nil {
		log.Error("Failed to convert the obj: %v", err)
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

func (tc *TFJobController) deletePdb(tfJob *tfv1alpha2.TFJob) error {
	
	// Check the pdb exist or not
	_, err := tc.kubeClientSet.PolicyV1beta1().PodDisruptionBudgets(tfjob.Namespace).Get(tfjob.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	tc.recorder.Event(tfJob, v1.EventTypeNormal, terminatedTFJobReason,
		"TFJob is terminated, deleting pdb")

	msg := fmt.Sprintf("Deleting pdb %s", tfJob.Name)
	log.Info(msg)

	if err := tc.kubeClientSet.PolicyV1beta1().PodDisruptionBudgets(tfjob.Namespace).Delete(tfjob.Name, metav1.GetOptions{}); err != nil {
		tc.recorder.Event(tfJob, v1.EventTypeWarning, FailedDeletePdbReason, "Error deleting: %v", err)
		return fmt.Errorf("unable to delete pdb: %v", err)
	} else {
		tc.recorder.Event(tfJob, v1.EventTypeNormal, SuccessfulDeletePdbReason, "Deleted pdb: %v", tfjob.Name)
	}

	return nil
}

func (tc *TFJobController) deletePodsAndServices(tfJob *tfv1alpha2.TFJob, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}
	tc.recorder.Event(tfJob, v1.EventTypeNormal, terminatedTFJobReason,
		"TFJob is terminated, deleting pods and services")

	// Delete nothing when the cleanPodPolicy is None.
	if *tfJob.Spec.CleanPodPolicy == tfv1alpha2.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if *tfJob.Spec.CleanPodPolicy == tfv1alpha2.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning {
			continue
		}
		if err := tc.podControl.DeletePod(pod.Namespace, pod.Name, tfJob); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := tc.serviceControl.DeleteService(pod.Namespace, pod.Name, tfJob); err != nil {
			return err
		}
	}
	return nil
}
