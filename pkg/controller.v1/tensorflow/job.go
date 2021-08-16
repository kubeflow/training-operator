package tensorflow

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	"github.com/kubeflow/common/pkg/util/k8sutil"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	failedMarshalTFJobReason  = "InvalidTFJobSpec"
	FailedDeleteJobReason     = "FailedDeleteJob"
	SuccessfulDeleteJobReason = "SuccessfulDeleteJob"
)

var (
	tfJobsCreatedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_tfjobs_created_total",
			Help: "Counts number of TF jobs created",
		},
		[]string{"job_namespace"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(tfJobsCreatedCount)
}

// DeleteJob implements ControllerInterface interface.
func (tc *TFController) DeleteJob(job interface{}) error {
	tfJob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfJob)
	}

	log := commonutil.LoggerForJob(tfJob)
	if err := tc.tfJobClientSet.KubeflowV1().TFJobs(tfJob.Namespace).Delete(context.TODO(), tfJob.Name, metav1.DeleteOptions{}); err != nil {
		tc.JobController.Recorder.Eventf(tfJob, v1.EventTypeWarning, FailedDeleteJobReason, "Error deleting: %v", err)
		log.Errorf("failed to delete job %s/%s, %v", tfJob.Namespace, tfJob.Name, err)
		return err
	}

	tc.JobController.Recorder.Eventf(tfJob, v1.EventTypeNormal, SuccessfulDeleteJobReason, "Deleted job: %v", tfJob.Name)
	log.Infof("job %s/%s has been deleted", tfJob.Namespace, tfJob.Name)
	return nil
}

// addTFJob sets the defaults and enqueue the current tfjob.
func (tc *TFController) addTFJob(obj interface{}) {
	// Convert from unstructured object.
	tfJob, err := tfJobFromUnstructured(obj)
	if err != nil {
		un, ok := obj.(*metav1unstructured.Unstructured)
		logger := &log.Entry{}
		if ok {
			logger = commonutil.LoggerForUnstructured(un, tfv1.Kind)
		}
		logger.Errorf("Failed to convert the TFJob: %v", err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to marshal the object to TFJob; the spec is invalid: %v", err)
			logger.Warn(errMsg)
			// TODO(jlewi): v1 doesn't appear to define an error type.
			tc.Recorder.Event(un, v1.EventTypeWarning, failedMarshalTFJobReason, errMsg)

			status := commonv1.JobStatus{
				Conditions: []commonv1.JobCondition{
					{
						Type:               commonv1.JobFailed,
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

			client, err := k8sutil.NewCRDRestClient(&tfv1.GroupVersion)

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
	// TODO(Jeffwan): Consider to change to scheme https://github.com/kubeflow/tf-operator/issues/1317#issuecomment-890397705
	tfv1.SetDefaults_TFJob(tfJob)
	scheme.Scheme.Default(tfJob)

	msg := fmt.Sprintf("TFJob %s is created.", tfJob.Name)
	logger := commonutil.LoggerForJob(tfJob)
	logger.Info(msg)

	// Add a created condition.
	err = commonutil.UpdateJobConditions(&tfJob.Status, commonv1.JobCreated, tfJobCreatedReason, msg)
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
	tfJobsCreatedCount.WithLabelValues(tfJob.Namespace).Inc()
}

// updateTFJob enqueues the current tfjob.
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
		curTFJobADS := curTFJob.Spec.RunPolicy.ActiveDeadlineSeconds
		if curTFJobADS == nil {
			return
		}
		oldTFJobADS := oldTFJob.Spec.RunPolicy.ActiveDeadlineSeconds
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
