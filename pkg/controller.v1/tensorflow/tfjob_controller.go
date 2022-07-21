// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tensorflow

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	commonutil "github.com/kubeflow/common/pkg/util"
	train_util "github.com/kubeflow/common/pkg/util/train"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	trainingoperatorcommon "github.com/kubeflow/training-operator/pkg/common"
	"github.com/kubeflow/training-operator/pkg/common/util"
)

const (
	// tfJobSucceededReason is added in a tfjob when it is succeeded.
	tfJobSucceededReason = "TFJobSucceeded"
	// tfJobRunningReason is added in a tfjob when it is running.
	tfJobRunningReason = "TFJobRunning"
	// tfJobFailedReason is added in a tfjob when it is failed.
	tfJobFailedReason = "TFJobFailed"
	// tfJobRestarting is added in a tfjob when it is restarting.
	tfJobRestartingReason = "TFJobRestarting"

	FailedDeleteJobReason     = "FailedDeleteJob"
	SuccessfulDeleteJobReason = "SuccessfulDeleteJob"

	controllerName = "tfjob-controller"

	// labels for pods and servers.
	tfReplicaTypeLabel  = "replica-type"
	tfReplicaIndexLabel = "replica-index"
	// volcanoTaskSpecKey task spec key used in pod annotation when EnableGangScheduling is true
	volcanoTaskSpecKey = "volcano.sh/task-spec"

	// gang scheduler name.
	gangSchedulerName = "volcano"
	// tfConfig is the environment variable name of TensorFlow cluster spec.
	tfConfig = "TF_CONFIG"
	// exitedWithCodeReason is the normal reason when the pod is exited because of the exit code.
	exitedWithCodeReason = "ExitedWithCode"
	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
	// gangSchedulingPodGroupAnnotation is the annotation key used by batch schedulers
	gangSchedulingPodGroupAnnotation = "scheduling.k8s.io/group-name"
)

func NewReconciler(mgr manager.Manager, enableGangScheduling bool) *TFJobReconciler {
	r := &TFJobReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor(controllerName),
		apiReader: mgr.GetAPIReader(),
		Log:       log.Log,
	}

	cfg := mgr.GetConfig()
	kubeClientSet := kubeclientset.NewForConfigOrDie(cfg)
	volcanoClientSet := volcanoclient.NewForConfigOrDie(cfg)
	sharedInformers := informers.NewSharedInformerFactory(kubeClientSet, 0)
	priorityClassInformer := sharedInformers.Scheduling().V1beta1().PriorityClasses()

	r.JobController = common.JobController{
		Controller:                  r,
		Expectations:                expectation.NewControllerExpectations(),
		Config:                      common.JobControllerConfiguration{EnableGangScheduling: enableGangScheduling},
		WorkQueue:                   &util.FakeWorkQueue{},
		Recorder:                    r.recorder,
		KubeClientSet:               kubeClientSet,
		VolcanoClientSet:            volcanoClientSet,
		PriorityClassLister:         priorityClassInformer.Lister(),
		PriorityClassInformerSynced: priorityClassInformer.Informer().HasSynced,
		PodControl:                  control.RealPodControl{KubeClient: kubeClientSet, Recorder: r.recorder},
		ServiceControl:              control.RealServiceControl{KubeClient: kubeClientSet, Recorder: r.recorder},
	}

	return r
}

// TFJobReconciler reconciles a TFJob object
type TFJobReconciler struct {
	common.JobController
	client.Client
	Scheme    *runtime.Scheme
	recorder  record.EventRecorder
	apiReader client.Reader
	Log       logr.Logger
}

//+kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeflow.org,resources=tfjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TFJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := r.Log.WithValues(kubeflowv1.TFJobSingular, req.NamespacedName)

	tfjob := &kubeflowv1.TFJob{}
	err := r.Get(ctx, req.NamespacedName, tfjob)
	if err != nil {
		logger.Info(err.Error(), "unable to fetch TFJob", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = kubeflowv1.ValidateV1TFJobSpec(&tfjob.Spec); err != nil {
		logger.Info(err.Error(), "TFJob failed validation", req.NamespacedName.String())
	}

	// Check if reconciliation is needed
	jobKey, err := common.KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get jobKey for job object %#v: %v", tfjob, err))
	}

	replicaTypes := util.GetReplicaTypes(tfjob.Spec.TFReplicaSpecs)
	needReconcile := util.SatisfiedExpectations(r.Expectations, jobKey, replicaTypes)

	if !needReconcile || tfjob.GetDeletionTimestamp() != nil {
		logger.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needReconcile, "deleted", tfjob.GetDeletionTimestamp() != nil)
		return ctrl.Result{}, nil
	}

	// Set default priorities to tfjob
	r.Scheme.Default(tfjob)

	// Use common to reconcile the job related pod and service
	err = r.ReconcileJobs(tfjob, tfjob.Spec.TFReplicaSpecs, tfjob.Status, &tfjob.Spec.RunPolicy)
	if err != nil {
		logrus.Warnf("Reconcile Tensorflow Job error %v", err)
		return ctrl.Result{}, err
	}

	t, err := util.DurationUntilExpireTime(&tfjob.Spec.RunPolicy, tfjob.Status)
	if err != nil {
		logrus.Warnf("Reconcile Tensorflow Job error %v", err)
		return ctrl.Result{}, err
	}
	if t >= 0 {
		return ctrl.Result{Requeue: true, RequeueAfter: t}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TFJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(r.ControllerName(), mgr, controller.Options{
		Reconciler: r,
	})

	if err != nil {
		return err
	}

	// using onOwnerCreateFunc is easier to set defaults
	if err = c.Watch(&source.Kind{Type: &kubeflowv1.TFJob{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{CreateFunc: r.onOwnerCreateFunc()},
	); err != nil {
		return err
	}

	// inject watching for job related pod
	if err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubeflowv1.TFJob{},
	}, predicate.Funcs{
		CreateFunc: util.OnDependentCreateFunc(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFunc(&r.JobController),
		DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
	}); err != nil {
		return err
	}

	// inject watching for job related service
	if err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubeflowv1.TFJob{},
	}, predicate.Funcs{
		CreateFunc: util.OnDependentCreateFunc(r.Expectations),
		UpdateFunc: util.OnDependentUpdateFunc(&r.JobController),
		DeleteFunc: util.OnDependentDeleteFunc(r.Expectations),
	}); err != nil {
		return err
	}

	return nil
}

func (r *TFJobReconciler) ControllerName() string {
	return controllerName
}

func (r *TFJobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return kubeflowv1.GroupVersion.WithKind(kubeflowv1.TFJobKind)
}

func (r *TFJobReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return kubeflowv1.GroupVersion
}

func (r *TFJobReconciler) GetGroupNameLabelValue() string {
	return kubeflowv1.GroupVersion.Group
}

func (r *TFJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	tfjob := &kubeflowv1.TFJob{}
	err := r.Get(context.Background(), types.NamespacedName{
		Namespace: namespace, Name: name,
	}, tfjob)
	return tfjob, err
}

func (r *TFJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &kubeflowv1.TFJob{}

	err := r.apiReader.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Error(err, "tensorflow job not found", "namespace", namespace, "name", name)
		} else {
			logrus.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// GetPodsForJob returns the set of pods that this job should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned Pods are pointers into the cache.
func (r *TFJobReconciler) GetPodsForJob(jobObject interface{}) ([]*corev1.Pod, error) {
	job, ok := jobObject.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("job is not of type metav1.Object")
	}

	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.GenLabels(job.GetName()),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist,
		client.MatchingLabelsSelector{Selector: selector}, client.InNamespace(job.GetNamespace()))
	if err != nil {
		return nil, err
	}

	pods := util.ConvertPodList(podlist.Items)

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := common.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := r.Controller.GetJobFromAPIClient(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := control.NewPodControllerRefManager(r.PodControl, job, selector, r.Controller.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimPods(pods)
}

// GetServicesForJob returns the set of services that this job should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned services are pointers into the cache.
func (r *TFJobReconciler) GetServicesForJob(jobObject interface{}) ([]*corev1.Service, error) {
	job, ok := jobObject.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("job is not of type metav1.Object")
	}

	// Create selector
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.GenLabels(job.GetName()),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all services to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	svclist := &corev1.ServiceList{}
	err = r.List(context.Background(), svclist,
		client.MatchingLabelsSelector{Selector: selector}, client.InNamespace(job.GetNamespace()))
	if err != nil {
		return nil, fmt.Errorf("couldn't get Service: %v", err)
	}

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing services (see #42639).
	canAdoptFunc := common.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := r.GetJobFromInformerCache(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := control.NewServiceControllerRefManager(r.ServiceControl, job, selector, r.Controller.GetAPIGroupVersionKind(), canAdoptFunc)

	services := util.ConvertServiceList(svclist.Items)
	return cm.ClaimServices(services)
}

func (r *TFJobReconciler) DeleteJob(job interface{}) error {
	tfJob, ok := job.(*kubeflowv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfJob)
	}

	log := commonutil.LoggerForJob(tfJob)
	if err := r.Delete(context.Background(), tfJob); err != nil {
		r.recorder.Eventf(tfJob, v1.EventTypeWarning, FailedDeleteJobReason, "Error deleting: %v", err)
		log.Errorf("failed to delete job %s/%s, %v", tfJob.Namespace, tfJob.Name, err)
		return err
	}

	r.recorder.Eventf(tfJob, v1.EventTypeNormal, SuccessfulDeleteJobReason, "Deleted job: %v", tfJob.Name)
	log.Infof("job %s/%s has been deleted", tfJob.Namespace, tfJob.Name)
	trainingoperatorcommon.DeletedJobsCounterInc(tfJob.Namespace, kubeflowv1.TFJobFrameworkName)
	return nil
}

func (r *TFJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	tfJob, ok := job.(*kubeflowv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfJob)
	}

	tfJobKey, err := common.KeyFunc(tfJob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfJob, err))
		return err
	}

	logger := commonutil.LoggerForJob(tfJob)

	worker0Completed, err := r.IsWorker0Completed(tfJob, replicas)
	if err != nil {
		logger.Warnf("check if worker 0 completed error %v", err)
		return err
	}

	// Set StartTime.
	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
		// enqueue a sync to check if job past ActiveDeadlineSeconds
		if tfJob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
			logger.Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *tfJob.Spec.RunPolicy.ActiveDeadlineSeconds)
			// TODO(Jeffwan): requeue job key in reconciler scenarios
			r.WorkQueue.AddAfter(tfJobKey, time.Duration(*tfJob.Spec.RunPolicy.ActiveDeadlineSeconds)*time.Second)
		}
	}
	// iterate the replica spec based on this order
	allTypes := []commonv1.ReplicaType{
		kubeflowv1.TFJobReplicaTypeChief,
		kubeflowv1.TFJobReplicaTypeEval,
		kubeflowv1.TFJobReplicaTypeMaster,
		kubeflowv1.TFJobReplicaTypePS,
		kubeflowv1.TFJobReplicaTypeWorker,
	}
	for _, rtype := range allTypes {
		if replicas[rtype] == nil {
			continue
		}
		spec := replicas[rtype]
		status := jobStatus.ReplicaStatuses[rtype]

		// Expect to have `replicas - succeeded` pods alive.
		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logger.Infof("TFJob=%s/%s, ReplicaType=%s expected=%d, running=%d, failed=%d",
			tfJob.Namespace, tfJob.Name, rtype, expected, running, failed)

		// If the TFJob contains Chief or Master spec, then we will update the status
		// according to the Chief/Master spec.
		if ContainsChiefOrMasterSpec(tfJob.Spec.TFReplicaSpecs) {
			if kubeflowv1.IsChieforMaster(rtype) {
				if running > 0 {
					msg := fmt.Sprintf("TFJob %s/%s is running.",
						tfJob.Namespace, tfJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus,
						commonv1.JobRunning, tfJobRunningReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof(
							"Append tfjob condition error: %v", err)
						return err
					}
				}
				if expected == 0 {
					msg := fmt.Sprintf("TFJob %s/%s successfully completed.",
						tfJob.Namespace, tfJob.Name)
					r.recorder.Event(tfJob, corev1.EventTypeNormal, tfJobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus,
						commonv1.JobSucceeded, tfJobSucceededReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
						return err
					}
					trainingoperatorcommon.SuccessfulJobsCounterInc(tfJob.Namespace, kubeflowv1.TFJobFrameworkName)
				}
			}
		} else {
			if rtype == kubeflowv1.TFJobReplicaTypeWorker {
				// Leave a succeeded condition for the following two cases:
				// 1. If default success policy is used and worker 0 has completed.
				// 2. If `SuccessPolicyAllWorkers` success policy is used and all workers are succeeded.
				if expected == 0 || (worker0Completed && *tfJob.Spec.SuccessPolicy != kubeflowv1.SuccessPolicyAllWorkers) {
					msg := fmt.Sprintf("TFJob %s/%s successfully completed.",
						tfJob.Namespace, tfJob.Name)
					r.recorder.Event(tfJob, corev1.EventTypeNormal, tfJobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus,
						commonv1.JobSucceeded, tfJobSucceededReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
						return err
					}
					trainingoperatorcommon.SuccessfulJobsCounterInc(tfJob.Namespace, kubeflowv1.TFJobFrameworkName)
				} else if running > 0 {
					// Some workers are still running, leave a running condition.
					msg := fmt.Sprintf("TFJob %s/%s is running.",
						tfJob.Namespace, tfJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, tfJobRunningReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
						return err
					}
				}
			}
		}

		if failed > 0 {
			restart := false
			for _, condition := range jobStatus.Conditions {
				if condition.Type == commonv1.JobRestarting {
					restart = true
				}
			}

			if restart {
				// job is restarting, no need to set it failed
				// we know it because we update the status condition when reconciling the replicas
				trainingoperatorcommon.RestartedJobsCounterInc(tfJob.Namespace, kubeflowv1.TFJobFrameworkName)
			} else {
				if tfJob.Spec.EnableDynamicWorker && rtype == kubeflowv1.TFJobReplicaTypeWorker {
					commonutil.LoggerForJob(tfJob).Infof("TFJob %s/%s continues regardless %d Worker replica(s) failed as enableDynamicWorker is set true.",
						tfJob.Namespace, tfJob.Name, failed)
					continue
				}
				msg := fmt.Sprintf("TFJob %s/%s has failed because %d %s replica(s) failed.",
					tfJob.Namespace, tfJob.Name, failed, rtype)
				r.recorder.Event(tfJob, corev1.EventTypeNormal, tfJobFailedReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus,
					commonv1.JobFailed, tfJobFailedReason, msg)
				if err != nil {
					commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
					return err
				}
				trainingoperatorcommon.FailedJobsCounterInc(tfJob.Namespace, kubeflowv1.TFJobFrameworkName)
			}
		}
	}
	// we assign the jobStatus to the tfJob.Status for testing purpose
	// it won't effect the main reconcile logic
	// because we already use oldStatus := jobStatus.DeepCopy() to record the oldStatus
	// and use !reflect.DeepEqual(*oldStatus, jobStatus) to decide whether to update the tfJob or not
	tfJob.Status = *jobStatus.DeepCopy()

	return nil
}

func (r *TFJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = map[commonv1.ReplicaType]*commonv1.ReplicaStatus{}
	}

	tfJob, ok := job.(*kubeflowv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfJob)
	}

	startTime := time.Now()
	logger := commonutil.LoggerForJob(tfJob)
	defer func() {
		logger.Infof("Finished updating TFJobs Status %q (%v)",
			tfJob.Name, time.Since(startTime))
	}()

	tfJob = tfJob.DeepCopy()
	tfJob.Status = *jobStatus.DeepCopy()

	result := r.Status().Update(context.Background(), tfJob)

	if result != nil {
		r.Log.WithValues("tfjob", types.NamespacedName{
			Namespace: tfJob.GetNamespace(),
			Name:      tfJob.GetName(),
		})
		return result
	}

	return nil
}

// Same as Func (tc *TFController) SetClusterSpec(...) in pod.go
func (r *TFJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	tfjob, ok := job.(*kubeflowv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfjob)
	}

	// Do not set TF_CONFIG for local training jobs.
	if !isDistributed(tfjob) {
		return nil
	}
	// Generate TF_CONFIG JSON string.
	tfConfigStr, err := genTFConfigJSONStr(tfjob, rtype, index)
	if err != nil {
		return err
	}

	if tfConfigStr == "" {
		return nil
	}
	// Add TF_CONFIG environment variable to tensorflow container in the pod.
	for i := range podTemplate.Spec.Containers {
		if podTemplate.Spec.Containers[i].Name == kubeflowv1.TFJobDefaultContainerName {
			if len(podTemplate.Spec.Containers[i].Env) == 0 {
				podTemplate.Spec.Containers[i].Env = make([]corev1.EnvVar, 0)
			}
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  tfConfig,
				Value: tfConfigStr,
			})
			break
		}
	}
	return nil
}

func (r *TFJobReconciler) GetDefaultContainerName() string {
	return kubeflowv1.TFJobDefaultContainerName
}

func (r *TFJobReconciler) GetDefaultContainerPortName() string {
	return kubeflowv1.TFJobDefaultPortName
}

func (r *TFJobReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	if ContainsChiefOrMasterSpec(replicas) {
		return rtype == kubeflowv1.TFJobReplicaTypeChief || rtype == kubeflowv1.TFJobReplicaTypeMaster
	}
	// else check if it is worker with index 0
	return rtype == kubeflowv1.TFJobReplicaTypeWorker && index == 0
}

// IsWorker0Completed returns true if pod of worker0 succeeded and exited with 0
func (r *TFJobReconciler) IsWorker0Completed(tfJob *kubeflowv1.TFJob, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) (bool, error) {
	worker0Completed := false
	_, ok := replicas[kubeflowv1.TFJobReplicaTypeWorker]
	if !ok {
		return true, nil
	}
	podSlices, err := r.getPodSlices(tfJob, replicas[kubeflowv1.TFJobReplicaTypeWorker].Replicas)
	if err != nil {
		return false, err
	}
	for index, podSlice := range podSlices {
		if len(podSlice) == 1 {
			pod := podSlice[0]
			exitCode := getContainerExitCode(pod)
			if index == 0 && exitCode == 0 && pod.Status.Phase == v1.PodSucceeded {
				worker0Completed = true
			}
		}
	}
	return worker0Completed, nil
}

// getPodSlices returns a slice, which element is the slice of pod.
// It gives enough information to caller to make decision to up/down scale resources.
func (r *TFJobReconciler) getPodSlices(tfjob *kubeflowv1.TFJob, replicasNum *int32) ([][]*v1.Pod, error) {
	logger := commonutil.LoggerForReplica(tfjob, strings.ToLower(string(kubeflowv1.TFJobReplicaTypeWorker)))

	pods, err := r.GetPodsForJob(tfjob)
	if err != nil {
		commonutil.LoggerForJob(tfjob).Warnf("getPodsForTFJob error %v", err)
		return nil, err
	}

	// Get all pods for the type rt.
	pods, err = r.JobController.FilterPodsForReplicaType(pods, strings.ToLower(string(kubeflowv1.TFJobReplicaTypeWorker)))
	if err != nil {
		return nil, err
	}

	podSlices := r.GetPodSlices(pods, int(*replicasNum), logger)
	return podSlices, nil
}

// In order to minimize the changes, we copy TFController's logic here to override kubeflow/commons reconcile logic
// This should be removed later unless TF has specific logics there
// reconcilePods checks and updates pods for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting pods.
func (r *TFJobReconciler) ReconcilePods(
	job interface{},
	jobStatus *commonv1.JobStatus,
	pods []*v1.Pod,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
) error {

	tfJob, ok := job.(*kubeflowv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfJob)
	}

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	logger := commonutil.LoggerForJob(tfJob)
	// Get all pods for the type rt.
	pods, err := r.FilterPodsForReplicaType(pods, rt)
	if err != nil {
		return err
	}
	numReplicas := int(*spec.Replicas)
	masterRole := false
	//restart := false
	//worker0Completed := false

	initializeReplicaStatuses(jobStatus, rtype)

	// GetPodSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have pods with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a pod with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], pod with replica-index 1 and 2 are out of range and will be deleted.
	podSlices := r.GetPodSlices(pods, numReplicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rt, index)
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rt, index)

			// check if this replica is the master role
			masterRole = r.IsMasterRole(replicas, rtype, index)
			// TODO: [should change to CreateNewPod]
			err = r.createNewPod(tfJob, rt, strconv.Itoa(index), spec, masterRole, replicas)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]

			// check if the index is in the valid range, if not, we should kill the pod
			if index < 0 || index >= numReplicas {
				err = r.PodControl.DeletePod(pod.Namespace, pod.Name, tfJob)
				if err != nil {
					return err
				}
			}
			// Get the exit code of the container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == r.GetDefaultContainerName() && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					logger.Infof("Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					r.Recorder.Eventf(tfJob, v1.EventTypeNormal, exitedWithCodeReason, "Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
				}
			}
			// Check if the pod is retryable.
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				if pod.Status.Phase == v1.PodFailed && train_util.IsRetryableExitCode(exitCode) {
					logger.Infof("Need to restart the pod: %v.%v", pod.Namespace, pod.Name)
					if err := r.PodControl.DeletePod(pod.Namespace, pod.Name, tfJob); err != nil {
						return err
					}

					// with common library framework, we have to handle restart status here
					// or we won't know which replica has been restarted in updateJobStatus after reconciling all replicas
					msg := fmt.Sprintf("TFJob %s is restarting because %s replica(s) failed.",
						tfJob.Name, rtype)
					r.Recorder.Event(tfJob, corev1.EventTypeWarning, tfJobRestartingReason, msg)
					err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, tfJobRestartingReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
						return err
					}
					trainingoperatorcommon.RestartedJobsCounterInc(tfJob.Namespace, kubeflowv1.TFJobFrameworkName)
				}
			}

			updateJobReplicaStatuses(jobStatus, rtype, pod)
		}
	}
	return nil
}

// createNewPod creates a new pod for the given index and type.
func (r *TFJobReconciler) createNewPod(tfjob *kubeflowv1.TFJob, rt, index string, spec *commonv1.ReplicaSpec, masterRole bool,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {

	tfjobKey, err := common.KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfjob, err))
		return err
	}
	expectationPodsKey := expectation.GenExpectationPodsKey(tfjobKey, rt)
	err = r.Expectations.ExpectCreations(expectationPodsKey, 1)
	if err != nil {
		return err
	}
	logger := commonutil.LoggerForReplica(tfjob, rt)
	// Create OwnerReference.
	controllerRef := r.GenOwnerReference(tfjob)

	// Set type and index for the worker.
	labels := r.GenLabels(tfjob.Name)
	labels[tfReplicaTypeLabel] = rt
	labels[tfReplicaIndexLabel] = index
	labels[commonv1.ReplicaTypeLabel] = rt
	labels[commonv1.ReplicaIndexLabel] = index

	if masterRole {
		labels[commonv1.JobRoleLabel] = "master"
	}

	podTemplate := spec.Template.DeepCopy()

	// Set name for the template.
	podTemplate.Name = common.GenGeneralName(tfjob.Name, rt, index)

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	if err := r.SetClusterSpec(tfjob, podTemplate, rt, index); err != nil {
		return err
	}

	// Submit a warning event if the user specifies restart policy for
	// the pod template. We recommend to set it from the replica level.
	if podTemplate.Spec.RestartPolicy != v1.RestartPolicy("") {
		errMsg := "Restart policy in pod template will be overwritten by restart policy in replica spec"
		logger.Warning(errMsg)
		r.Recorder.Event(tfjob, v1.EventTypeWarning, podTemplateRestartPolicyReason, errMsg)
	}
	setRestartPolicy(podTemplate, spec)

	// if gang-scheduling is enabled:
	// 1. if user has specified other scheduler, we report a warning without overriding any fields.
	// 2. if no SchedulerName is set for pods, then we set the SchedulerName to "volcano".
	if r.Config.EnableGangScheduling {
		podSchedulerName := util.GetSchedulerName(replicas)
		if len(podSchedulerName) == 0 {
			podTemplate.Spec.SchedulerName = gangSchedulerName
		} else if strings.Compare(podSchedulerName, gangSchedulerName) != 0 {
			errMsg := "Another scheduler is specified when gang-scheduling is enabled and it will not be overwritten"
			logger.Warning(errMsg)
			r.Recorder.Event(tfjob, v1.EventTypeWarning, podTemplateSchedulerNameReason, errMsg)
		}

		if podTemplate.Annotations == nil {
			podTemplate.Annotations = map[string]string{}
		}
		podTemplate.Annotations[gangSchedulingPodGroupAnnotation] = tfjob.GetName()
		podTemplate.Annotations[volcanoTaskSpecKey] = rt
	}

	err = r.PodControl.CreatePodsWithControllerRef(tfjob.Namespace, podTemplate, tfjob, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Pod is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the pod keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// pod when the expectation expires.
		return nil
	} else if err != nil {
		// Decrement the expected number of creates because the informer won't observe this pod
		logger.Infof(
			"Failed creation, decrementing expectations for tfjob %s/%s, key %s",
			tfjob.Namespace, tfjob.Name, expectationPodsKey)
		r.Expectations.CreationObserved(expectationPodsKey)
		return err
	}
	return nil
}

// onOwnerCreateFunc modify creation condition.
func (r *TFJobReconciler) onOwnerCreateFunc() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		tfJob, ok := e.Object.(*kubeflowv1.TFJob)
		if !ok {
			return true
		}

		r.Scheme.Default(tfJob)
		msg := fmt.Sprintf("TFJob %s is created.", e.Object.GetName())
		logrus.Info(msg)
		trainingoperatorcommon.CreatedJobsCounterInc(tfJob.Namespace, kubeflowv1.TFJobFrameworkName)
		if err := commonutil.UpdateJobConditions(&tfJob.Status, commonv1.JobCreated, "TFJobCreated", msg); err != nil {
			log.Log.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
