package test_job

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	v1 "github.com/kubeflow/common/test_job/apis/test_job/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ commonv1.ControllerInterface = &TestJobController{}

type TestJobController struct {
	Job      *v1.TestJob
	Pods     []*corev1.Pod
	Services []*corev1.Service
}

func (t TestJobController) GetPodsForJob(job interface{}) ([]*corev1.Pod, error) {
	return []*corev1.Pod{}, nil
}

func (t TestJobController) GetServicesForJob(job interface{}) ([]*corev1.Service, error) {
	return []*corev1.Service{}, nil
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

func (TestJobController) GetJobRoleKey() string {
	return commonv1.JobRoleLabel
}

func (TestJobController) GetDefaultContainerPortName() string {
	return "default-port-name"
}

func (TestJobController) GetDefaultContainerPortNumber() int32 {
	return int32(9999)
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

func (t *TestJobController) CreateService(job interface{}, service *corev1.Service) error {
	return nil
}

func (t *TestJobController) DeleteService(job interface{}, name string, namespace string) error {
	log.Info("Deleting service " + name)
	var remainingServices []*corev1.Service
	for _, tservice := range t.Services {
		if tservice.Name != name {
			remainingServices = append(remainingServices, tservice)
		}
	}
	t.Services = remainingServices
	return nil
}

func (t *TestJobController) CreatePod(job interface{}, pod *corev1.Pod) error {
	return nil
}

func (t *TestJobController) DeletePod(job interface{}, pod *corev1.Pod) error {
	log.Info("Deleting pod " + pod.Name)
	var remainingPods []*corev1.Pod
	for _, tpod := range t.Pods {
		if tpod.Name != pod.Name {
			remainingPods = append(remainingPods, tpod)
		}
	}
	t.Pods = remainingPods
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
