package test_job

import (
	commonv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/common"
	v1 "github.com/kubeflow/training-operator/test_job/apis/test_job/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ common.ControllerInterface = &TestJobController{}

type TestJobController struct {
	common.ControllerInterface
	Job      *v1.TestJob
	Pods     []*corev1.Pod
	Services []*corev1.Service
}

func (TestJobController) ControllerName() string {
	return "test-operator"
}

func (TestJobController) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return v1.SchemeGroupVersionKind
}

func (TestJobController) GetAPIGroupVersion() schema.GroupVersion {
	return v1.SchemeGroupVersion
}

func (TestJobController) GetGroupNameLabelValue() string {
	return v1.GroupName
}

func (TestJobController) GetDefaultContainerPortName() string {
	return "default-port-name"
}

func (t *TestJobController) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	return t.Job, nil
}

func (t *TestJobController) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	return t.Job, nil
}

func (t *TestJobController) DeleteJob(job interface{}) error {
	log.Info("Delete job")
	t.Job = nil
	return nil
}

func (t *TestJobController) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	jobStatus *commonv1.JobStatus) error {
	return nil
}

func (t *TestJobController) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	return nil
}

func (t *TestJobController) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return nil
}

func (t *TestJobController) GetDefaultContainerName() string {
	return "default-container"
}

func (t *TestJobController) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool {
	return true
}
