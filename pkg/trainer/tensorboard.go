package trainer

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"mlkube.io/pkg/spec"
)

const TbPort = 6006

// TBReplicaSet represent the RS for the TensorBoard instance
type TBReplicaSet struct {
	ClientSet kubernetes.Interface
	Job       *TrainingJob
	Spec      spec.TensorBoardSpec
}

func NewTBReplicaSet(clientSet kubernetes.Interface, s spec.TensorBoardSpec, job *TrainingJob) (*TBReplicaSet, error) {
	if s.LogDir == "" {
		return nil, errors.New("tbReplicaSpec.LogDir must be specified")
	}

	return &TBReplicaSet{
		ClientSet: clientSet,
		Job:       job,
		Spec:      s,
	}, nil
}

func (s *TBReplicaSet) Create() error {
	// By default we assume TensorBoard's service will be a ClusterIP
	// unless specified otherwise by the user
	st := v1.ServiceType("ClusterIP")
	if s.Spec.ServiceType != "" {
		st = s.Spec.ServiceType
	}

	// create the service exposing TensorBoard
	service := &v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   s.jobName(),
			Labels: s.Labels(),
		},
		Spec: v1.ServiceSpec{
			Type:     st,
			Selector: s.Labels(),
			Ports: []v1.ServicePort{
				{
					Name: "tb-port",
					Port: 80,
					TargetPort: intstr.IntOrString{
						IntVal: TbPort,
					},
				},
			},
		},
	}

	log.Infof("Creating Service: %v", service.ObjectMeta.Name)
	_, err := s.ClientSet.CoreV1().Services(NAMESPACE).Create(service)

	// If the job already exists do nothing.
	if err != nil {
		if k8s_errors.IsAlreadyExists(err) {
			log.Infof("Service %v already exists.", s.jobName())
		} else {
			return err
		}
	}

	newD := &v1beta1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   s.jobName(),
			Labels: s.Labels(),
		},
		Spec: v1beta1.DeploymentSpec{
			Selector: &meta_v1.LabelSelector{
				MatchLabels: s.Labels(),
			},
			Replicas: proto.Int32(1),
			Template: s.getDeploymentSpecTemplate(),
		},
	}

	log.Infof("Creating Deployment: %v", newD.ObjectMeta.Name)
	_, err = s.ClientSet.ExtensionsV1beta1().Deployments(NAMESPACE).Create(newD)

	if err != nil {
		if k8s_errors.IsAlreadyExists(err) {
			log.Infof("%v already exists.", s.jobName())
		} else {
			return err
		}
	}
	return nil
}

func (s *TBReplicaSet) Delete() error {
	failures := false

	delProp := meta_v1.DeletePropagationForeground
	err := s.ClientSet.ExtensionsV1beta1().Deployments(NAMESPACE).Delete(s.jobName(), &meta_v1.DeleteOptions{
		PropagationPolicy: &delProp,
	})
	if err != nil {
		log.Errorf("There was a problem deleting TensorBoard's deployment %v; %v", s.jobName(), err)
		failures = true
	}

	err = s.ClientSet.CoreV1().Services(NAMESPACE).Delete(s.jobName(), &meta_v1.DeleteOptions{})
	if err != nil {
		log.Errorf("Error deleting service: %v; %v", s.jobName(), err)
		failures = true
	}

	if failures {
		return errors.New("There was an issue deleting TensorBoard's resources")
	}
	return nil
}

func (s *TBReplicaSet) getDeploymentSpecTemplate() v1.PodTemplateSpec {
	// TODO: make the TensorFlow image a parameter of the job operator.
	c := &v1.Container{
		Name:  s.jobName(),
		Image: "tensorflow/tensorflow",
		Command: []string{
			"tensorboard", "--logdir", s.Spec.LogDir, "--host", "0.0.0.0",
		},
		Ports: []v1.ContainerPort{
			{
				ContainerPort: TbPort,
			},
		},
		VolumeMounts: make([]v1.VolumeMount, 0),
	}

	for _, v := range s.Spec.VolumeMounts {
		c.VolumeMounts = append(c.VolumeMounts, v)
	}

	ps := &v1.PodSpec{
		Containers: []v1.Container{*c},
		Volumes:    make([]v1.Volume, 0),
	}

	for _, v := range s.Spec.Volumes {
		ps.Volumes = append(ps.Volumes, v)
	}

	return v1.PodTemplateSpec{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   s.jobName(),
			Labels: s.Labels(),
		},
		Spec: *ps,
	}

}

func (s *TBReplicaSet) Labels() KubernetesLabels {
	return KubernetesLabels(map[string]string{
		"mlkube.io":  "",
		"runtime_id": s.Job.job.Spec.RuntimeId,
		"app":        "tensorboard",
	})
}

func (s *TBReplicaSet) jobName() string {
	return fmt.Sprintf("tensorboard-%v", strings.ToLower(s.Job.job.Spec.RuntimeId))
}
