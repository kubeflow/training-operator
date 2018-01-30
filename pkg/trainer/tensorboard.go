package trainer

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/tensorflow/k8s/pkg/apis/tensorflow/helper"
	tfv1alpha1 "github.com/tensorflow/k8s/pkg/apis/tensorflow/v1alpha1"
)

const TbPort = 6006

// TBReplicaSet represent the RS for the TensorBoard instance
type TBReplicaSet struct {
	ClientSet kubernetes.Interface
	Job       *TrainingJob
	Spec      tfv1alpha1.TensorBoardSpec
}

func NewTBReplicaSet(clientSet kubernetes.Interface, s tfv1alpha1.TensorBoardSpec, job *TrainingJob) (*TBReplicaSet, error) {
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
			OwnerReferences: []meta_v1.OwnerReference{
				helper.AsOwner(s.Job.job),
			},
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
	_, err := s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Create(service)

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
			OwnerReferences: []meta_v1.OwnerReference{
				helper.AsOwner(s.Job.job),
			},
		},
		Spec: v1beta1.DeploymentSpec{
			Selector: &meta_v1.LabelSelector{
				MatchLabels: s.Labels(),
			},
			Replicas: proto.Int32(1),
			Template: s.getDeploymentSpecTemplate(s.Job.job.Spec.TFImage),
		},
	}

	log.Infof("Creating Deployment: %v", newD.ObjectMeta.Name)
	_, err = s.ClientSet.ExtensionsV1beta1().Deployments(s.Job.job.ObjectMeta.Namespace).Create(newD)

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
	log.V(1).Infof("Deleting deployment %v:%v", s.Job.job.ObjectMeta.Namespace, s.jobName())
	err := s.ClientSet.ExtensionsV1beta1().Deployments(s.Job.job.ObjectMeta.Namespace).Delete(s.jobName(), &meta_v1.DeleteOptions{
		PropagationPolicy: &delProp,
	})
	if err != nil {
		log.Errorf("There was a problem deleting TensorBoard's deployment %v; %v", s.jobName(), err)
		failures = true
	}

	log.V(1).Infof("Deleting service %v:%v", s.Job.job.ObjectMeta.Namespace, s.jobName())
	err = s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Delete(s.jobName(), &meta_v1.DeleteOptions{})
	if err != nil {
		log.Errorf("Error deleting service: %v; %v", s.jobName(), err)
		failures = true
	}

	if failures {
		return errors.New("There was an issue deleting TensorBoard's resources")
	}
	return nil
}

func (s *TBReplicaSet) getDeploymentSpecTemplate(image string) v1.PodTemplateSpec {
	// TODO: make the TensorFlow image a parameter of the job operator.
	c := &v1.Container{
		Name:  s.jobName(),
		Image: image,
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

	c.VolumeMounts = append(c.VolumeMounts, s.Spec.VolumeMounts...)

	ps := &v1.PodSpec{
		Containers: []v1.Container{*c},
		Volumes:    make([]v1.Volume, 0),
	}

	ps.Volumes = append(ps.Volumes, s.Spec.Volumes...)

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
		"kubeflow.org": "",
		"runtime_id":   s.Job.job.Spec.RuntimeId,
		"app":          "tensorboard",
		"tf_job_name":  s.Job.job.ObjectMeta.Name,
	})
}

func (s *TBReplicaSet) jobName() string {
	// Truncate tfjob name to 40 characters
	// The whole job name should be compliant with the DNS_LABEL spec, up to a max length of 63 characters
	// Thus jobname(40 chars)-tensorboard(11 chars)-runtimeId(4 chars), also leaving some spaces
	// See https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
	return fmt.Sprintf("%v-tensorboard-%v", fmt.Sprintf("%.40s", s.Job.job.ObjectMeta.Name), strings.ToLower(s.Job.job.Spec.RuntimeId))
}
