package test_job

import (
	"context"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	commonreconciler "github.com/kubeflow/training-operator/pkg/reconciler.v1/common"
	v1 "github.com/kubeflow/training-operator/test_job/apis/test_job/v1"
	"github.com/kubeflow/training-operator/test_job/client/clientset/versioned/scheme"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TestReconciler struct {
	commonreconciler.ReconcilerUtil
	commonreconciler.ServiceReconciler
	commonreconciler.PodReconciler
	commonreconciler.VolcanoReconciler
	commonreconciler.JobReconciler

	FC       client.Client
	Job      *v1.TestJob
	Pods     []*corev1.Pod
	Services []*corev1.Service
	PodGroup client.Object
}

func NewTestReconciler() *TestReconciler {
	scm := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scm))
	utilruntime.Must(v1.AddToScheme(scm))

	fakeClient := fake.NewClientBuilder().WithScheme(scm).Build()

	r := &TestReconciler{
		FC: fakeClient,
	}

	// Generate Bare Components
	jobR := commonreconciler.BareJobReconciler(fakeClient)
	jobR.OverrideForJobInterface(r, r, r, r)

	podR := commonreconciler.BarePodReconciler(fakeClient)
	podR.OverrideForPodInterface(r, r, r)

	svcR := commonreconciler.BareServiceReconciler(fakeClient)
	svcR.OverrideForServiceInterface(r, r, r)

	gangR := commonreconciler.BareVolcanoReconciler(fakeClient, nil, false)
	gangR.OverrideForGangSchedulingInterface(r)

	Log := log.Log
	utilR := commonreconciler.BareUtilReconciler(nil, Log, scm)
	//kubeflowReconciler := commonreconciler.BareKubeflowReconciler()

	r.JobReconciler = *jobR
	r.PodReconciler = *podR
	r.ServiceReconciler = *svcR
	r.VolcanoReconciler = *gangR
	r.ReconcilerUtil = *utilR

	return r
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	job, err := r.GetJob(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger := r.GetLogger(job)

	if job.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	scheme.Scheme.Default(job)

	// Get rid of SatisfiedExpectation
	replicasSpec, err := r.ExtractReplicasSpec(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	runPolicy, err := r.ExtractRunPolicy(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	status, err := r.ExtractJobStatus(job)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.ReconcileJob(ctx, job, replicasSpec, status, runPolicy)
	if err != nil {
		logger.Info("Reconcile Test Job error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TestReconciler) GetReconcilerName() string {
	return "Test Reconciler"
}

func (r *TestReconciler) GetJob(ctx context.Context, req ctrl.Request) (client.Object, error) {
	return r.Job, nil
}

func (r *TestReconciler) GetDefaultContainerName() string {
	return v1.DefaultContainerName
}

func (r *TestReconciler) GetPodGroupForJob(ctx context.Context, job client.Object) (client.Object, error) {
	return r.PodGroup, nil
}

func (r *TestReconciler) GetPodsForJob(ctx context.Context, job client.Object) ([]*corev1.Pod, error) {
	return r.Pods, nil
}

func (r *TestReconciler) GetServicesForJob(ctx context.Context, job client.Object) ([]*corev1.Service, error) {
	return r.Services, nil
}

func (r *TestReconciler) ExtractReplicasSpec(job client.Object) (map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec, error) {
	tj := job.(*v1.TestJob)

	rs := map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec{}
	for k, v := range tj.Spec.TestReplicaSpecs {
		rs[kubeflowv1.ReplicaType(k)] = v
	}

	return rs, nil
}

func (r *TestReconciler) ExtractRunPolicy(job client.Object) (*kubeflowv1.RunPolicy, error) {
	tj := job.(*v1.TestJob)

	return tj.Spec.RunPolicy, nil
}

func (r *TestReconciler) ExtractJobStatus(job client.Object) (*kubeflowv1.JobStatus, error) {
	tj := job.(*v1.TestJob)

	return &tj.Status, nil
}

func (r *TestReconciler) IsMasterRole(replicas map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec, rtype kubeflowv1.ReplicaType, index int) bool {
	return string(rtype) == string(v1.TestReplicaTypeMaster)
}
