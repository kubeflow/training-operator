package tensorflow

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
	"github.com/kubeflow/tf-operator/pkg/util/k8sutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	failedMarshalTFJobReason = "InvalidTFJobSpec"
)

var (
	tfJobsCreatedCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tf_operator_jobs_created_total",
		Help: "Counts number of TF jobs created",
	})
)

// When a pod is added, set the defaults and enqueue the current tfjob.
func (tc *TFController) addTFJob(obj interface{}) {
	// Convert from unstructured object.
	tfJob, err := tfJobFromUnstructured(obj)
	if err != nil {
		un, ok := obj.(*metav1unstructured.Unstructured)
		logger := &log.Entry{}
		if ok {
			logger = tflogger.LoggerForUnstructured(un, tfv1.Kind)
		}
		logger.Errorf("Failed to convert the TFJob: %v", err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to marshal the object to TFJob; the spec is invalid: %v", err)
			logger.Warn(errMsg)
			// TODO(jlewi): v1 doesn't appear to define an error type.
			tc.Recorder.Event(un, v1.EventTypeWarning, failedMarshalTFJobReason, errMsg)

			status := common.JobStatus{
				Conditions: []common.JobCondition{
					{
						Type:               common.JobFailed,
						Status:             v1.ConditionTrue,
						LastUpdateTime:     metav1.Now(),
						LastTransitionTime: metav1.Now(),
						Reason:             failedMarshalTFJobReason,
						Message:            errMsg,
					},
				},
			}

			statusMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&status)

			if err != nil {
				logger.Errorf("Could not covert the TFJobStatus to unstructured; %v", err)
				return
			}

			client, err := k8sutil.NewCRDRestClient(&tfv1.SchemeGroupVersion)

			if err == nil {
				if err1 := metav1unstructured.SetNestedField(un.Object, statusMap, "status"); err1 != nil {
					logger.Errorf("Could not set nested field: %v", err1)
				}
				logger.Infof("Updating the job to: %+v", un.Object)
				err = client.UpdateStatus(un, tfv1.Plural)
				if err != nil {
					logger.Errorf("Could not update the TFJob: %v", err)
				}
			} else {
				logger.Errorf("Could not create a REST client to update the TFJob")
			}
		}
		return
	}

	// Set default for the new tfjob.
	scheme.Scheme.Default(tfJob)

	msg := fmt.Sprintf("TFJob %s is created.", tfJob.Name)
	logger := tflogger.LoggerForJob(tfJob)
	logger.Info(msg)

	// Add a created condition.
	err = updateTFJobConditions(tfJob, common.JobCreated, tfJobCreatedReason, msg)
	if err != nil {
		logger.Errorf("Append tfJob condition error: %v", err)
		return
	}

	// Convert from tfjob object
	err = unstructuredFromTFJob(obj, tfJob)
	if err != nil {
		logger.Errorf("Failed to convert the obj: %v", err)
		return
	}
	tc.enqueueTFJob(obj)
	tfJobsCreatedCount.Inc()
}

// When a pod is updated, enqueue the current tfjob.
func (tc *TFController) updateTFJob(old, cur interface{}) {
	oldTFJob, err := tfJobFromUnstructured(old)
	if err != nil {
		return
	}
	curTFJob, err := tfJobFromUnstructured(cur)
	if err != nil {
		return
	}

	// never return error
	key, err := KeyFunc(curTFJob)
	if err != nil {
		return
	}

	log.Infof("Updating tfjob: %s", oldTFJob.Name)
	tc.enqueueTFJob(cur)

	// check if need to add a new rsync for ActiveDeadlineSeconds
	if curTFJob.Status.StartTime != nil {
		curTFJobADS := curTFJob.Spec.ActiveDeadlineSeconds
		if curTFJobADS == nil {
			return
		}
		oldTFJobADS := oldTFJob.Spec.ActiveDeadlineSeconds
		if oldTFJobADS == nil || *oldTFJobADS != *curTFJobADS {
			now := metav1.Now()
			start := curTFJob.Status.StartTime.Time
			passed := now.Time.Sub(start)
			total := time.Duration(*curTFJobADS) * time.Second
			// AddAfter will handle total < passed
			tc.WorkQueue.AddAfter(key, total-passed)
			log.Infof("job ActiveDeadlineSeconds updated, will rsync after %d seconds", total-passed)
		}
	}
}

func (tc *TFController) deletePodsAndServices(tfJob *tfv1.TFJob, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}

	// Delete nothing when the cleanPodPolicy is None.
	if *tfJob.Spec.CleanPodPolicy == common.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if *tfJob.Spec.CleanPodPolicy == common.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning {
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

func (tc *TFController) cleanupTFJob(tfJob *tfv1.TFJob) error {
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

// deleteTFJob deletes the given TFJob.
func (tc *TFController) deleteTFJob(tfJob *tfv1.TFJob) error {
	return tc.tfJobClientSet.KubeflowV1().TFJobs(tfJob.Namespace).Delete(tfJob.Name, &metav1.DeleteOptions{})
}

func getTotalReplicas(tfjob *tfv1.TFJob) int32 {
	tfjobReplicas := int32(0)
	for _, r := range tfjob.Spec.TFReplicaSpecs {
		tfjobReplicas += *r.Replicas
	}
	return tfjobReplicas
}

func getTotalFailedReplicas(tfjob *tfv1.TFJob) int32 {
	totalFailedReplicas := int32(0)
	for rtype := range tfjob.Status.ReplicaStatuses {
		totalFailedReplicas += tfjob.Status.ReplicaStatuses[rtype].Failed
	}
	return totalFailedReplicas
}
