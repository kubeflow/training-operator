package tensorflow

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
)

const (
	failedMarshalTFJobReason = "FailedMarshalTFJob"
	terminatedTFJobReason    = "TFJobTerminated"
)

// When a pod is added, set the defaults and enqueue the current tfjob.
func (tc *TFController) addTFJob(obj interface{}) {
	// Convert from unstructured object.
	tfJob, err := tfJobFromUnstructured(obj)
	if err != nil {
		un, ok := obj.(*metav1unstructured.Unstructured)
		logger := &log.Entry{}
		if ok {
			logger = tflogger.LoggerForUnstructured(un, tfv1alpha2.Kind)
		}
		logger.Errorf("Failed to convert the TFJob: %v", err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to TFJob object: %v", err)
			logger.Warn(errMsg)
			tc.Recorder.Event(tfJob, v1.EventTypeWarning, failedMarshalTFJobReason, errMsg)
		}
		return
	}

	// Set default for the new tfjob.
	scheme.Scheme.Default(tfJob)

	msg := fmt.Sprintf("TFJob %s is created.", tfJob.Name)
	logger := tflogger.LoggerForJob(tfJob)
	logger.Info(msg)

	// Add a created condition.
	err = updateTFJobConditions(tfJob, tfv1alpha2.TFJobCreated, tfJobCreatedReason, msg)
	if err != nil {
		logger.Errorf("Append tfJob condition error: %v", err)
		return
	}

	// Convert from tfjob object
	err = unstructuredFromTFJob(obj, tfJob)
	if err != nil {
		logger.Error("Failed to convert the obj: %v", err)
		return
	}
	tc.enqueueTFJob(obj)
}

// When a pod is updated, enqueue the current tfjob.
func (tc *TFController) updateTFJob(old, cur interface{}) {
	oldTFJob, err := tfJobFromUnstructured(old)
	if err != nil {
		log.Errorf("failed to convert old tfjob from unstructured %v, %v", old, err)
		return
	}
	curTFJob, err := tfJobFromUnstructured(cur)
	if err != nil {
		log.Errorf("failed to convert cur tfjob from unstructured %v, %v", cur, err)
		return
	}
	if curTFJob.ResourceVersion == oldTFJob.ResourceVersion {
		// Periodic resync will send update events for all known TFJobs.
		// Two different versions of the same TFJob will always have different RVs
		return
	}
	log.Infof("Updating tfjob: %s", oldTFJob.Name)
	tc.enqueueTFJob(cur)
}

func (tc *TFController) deletePodsAndServices(tfJob *tfv1alpha2.TFJob, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}
	tc.Recorder.Event(tfJob, v1.EventTypeNormal, terminatedTFJobReason,
		"TFJob is terminated, deleting pods and services")

	// Delete nothing when the cleanPodPolicy is None.
	if *tfJob.Spec.CleanPodPolicy == tfv1alpha2.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if *tfJob.Spec.CleanPodPolicy == tfv1alpha2.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning {
			continue
		}
		if err := tc.PodControl.DeletePod(pod.Namespace, pod.Name, tfJob); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := tc.ServiceControl.DeleteService(pod.Namespace, pod.Name, tfJob); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TFController) cleanupTFJob(tfJob *tfv1alpha2.TFJob) error {
	currentTime := time.Now()
	ttl := tfJob.Spec.TTLSecondsAfterFinished
	if ttl == nil {
		// do nothing if the cleanup delay is not set
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if currentTime.After(tfJob.Status.CompletionTime.Add(duration)) {
		err := tc.deleteTFJobHandler(tfJob)
		if err != nil {
			tflogger.LoggerForJob(tfJob).Warnf("Cleanup TFJob error: %v.", err)
			return err
		}
		return nil
	}
	key, err := KeyFunc(tfJob)
	if err != nil {
		tflogger.LoggerForJob(tfJob).Warnf("Couldn't get key for tfjob object: %v", err)
		return err
	}
	tc.WorkQueue.AddRateLimited(key)
	return nil
}

// deleteTFJob delets the given TFJob.
func (tc *TFController) deleteTFJob(tfJob *tfv1alpha2.TFJob) error {
	return tc.tfJobClientSet.KubeflowV1alpha2().TFJobs(tfJob.Namespace).Delete(tfJob.Name, &metav1.DeleteOptions{})
}
