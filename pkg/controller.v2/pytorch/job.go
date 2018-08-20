package pytorch

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	v1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1alpha2"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
)

const (
	failedMarshalPyTorchJobReason = "FailedMarshalPyTorchJob"
	terminatedPyTorchJobReason    = "PyTorchJobTerminated"
)

// When a pod is added, set the defaults and enqueue the current pytorchjob.
func (pc *PyTorchController) addPyTorchJob(obj interface{}) {
	// Convert from unstructured object.
	job, err := jobFromUnstructured(obj)
	if err != nil {
		un, ok := obj.(*metav1unstructured.Unstructured)
		logger := &log.Entry{}
		if ok {
			logger = pylogger.LoggerForUnstructured(un, v1alpha2.Kind)
		}
		logger.Errorf("Failed to convert the PyTorchJob: %v", err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to PyTorchJob object: %v", err)
			logger.Warn(errMsg)
		}
		return
	}

	// Set default for the new job.
	scheme.Scheme.Default(job)

	msg := fmt.Sprintf("PyTorchJob %s is created.", job.Name)
	logger := pylogger.LoggerForJob(job)
	logger.Info(msg)

	// Add a created condition.
	err = updatePyTorchJobConditions(job, v1alpha2.PyTorchJobCreated, pytorchJobCreatedReason, msg)
	if err != nil {
		logger.Errorf("Append job condition error: %v", err)
		return
	}

	// Convert from pytorchjob object
	err = unstructuredFromPyTorchJob(obj, job)
	if err != nil {
		logger.Error("Failed to convert the obj: %v", err)
		return
	}
	pc.enqueuePyTorchJob(obj)
}

// When a pod is updated, enqueue the current pytorchjob.
func (pc *PyTorchController) updatePyTorchJob(old, cur interface{}) {
	oldPyTorchJob, err := jobFromUnstructured(old)
	if err != nil {
		return
	}
	log.Infof("Updating pytorchjob: %s", oldPyTorchJob.Name)
	pc.enqueuePyTorchJob(cur)
}

func (pc *PyTorchController) deletePodsAndServices(job *v1alpha2.PyTorchJob, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}
	pc.Recorder.Event(job, v1.EventTypeNormal, terminatedPyTorchJobReason,
		"PyTorchJob is terminated, deleting pods and services")

	// Delete nothing when the cleanPodPolicy is None.
	if *job.Spec.CleanPodPolicy == v1alpha2.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if *job.Spec.CleanPodPolicy == v1alpha2.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning {
			continue
		}
		if err := pc.PodControl.DeletePod(pod.Namespace, pod.Name, job); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := pc.ServiceControl.DeleteService(pod.Namespace, pod.Name, job); err != nil {
			return err
		}
	}
	return nil
}

func (pc *PyTorchController) cleanupPyTorchJob(job *v1alpha2.PyTorchJob) error {
	currentTime := time.Now()
	ttl := job.Spec.TTLSecondsAfterFinished
	if ttl == nil {
		// do nothing if the cleanup delay is not set
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if currentTime.After(job.Status.CompletionTime.Add(duration)) {
		err := pc.deletePyTorchJobHandler(job)
		if err != nil {
			pylogger.LoggerForJob(job).Warnf("Cleanup PyTorchJob error: %v.", err)
			return err
		}
		return nil
	}
	key, err := KeyFunc(job)
	if err != nil {
		pylogger.LoggerForJob(job).Warnf("Couldn't get key for pytorchjob object: %v", err)
		return err
	}
	pc.WorkQueue.AddRateLimited(key)
	return nil
}

// deletePyTorchJob delets the given PyTorchJob.
func (pc *PyTorchController) deletePyTorchJob(job *v1alpha2.PyTorchJob) error {
	return pc.jobClientSet.Pytorch().PyTorchJobs(job.Namespace).Delete(job.Name, &metav1.DeleteOptions{})
}
