package mxnet

import (
	"context"
	"encoding/json"
	"fmt"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	mxv1 "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	"github.com/kubeflow/tf-operator/pkg/common/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"strings"
)

const (
	// mxConfig is the environment variable name of MXNet cluster spec.
	mxConfig = "MX_CONFIG"
)

func (r *MXJobReconciler) GetPodsForJob(job interface{}) ([]*corev1.Pod, error) {
	mxJob, ok := job.(*mxv1.MXJob)
	if !ok {
		return nil, fmt.Errorf("%v is not a type of MXJob", mxJob)
	}

	//log := mxlogger.LoggerForJob(mxJob)
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.

	labelSelector := metav1.LabelSelector{MatchLabels: r.JobController.GenLabels(mxJob.Name)}
	opts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	podlist, err := r.KubeClientSet.CoreV1().Pods(mxJob.Namespace).List(context.Background(), opts)
	if err != nil {
		return nil, err
	}
	return util.ConvertPodList(podlist.Items), nil
}


func (r *MXJobReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	mxJob, ok := job.(*mxv1.MXJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", mxJob)
	}

	// Generate MX_CONFIG JSON.
	mxConfigData, err := genMXConfig(mxJob, rtype, index)
	if err != nil {
		return err
	}

	// Generate MX_CONFIG JSON Str.
	mxConfigJson, err := json.Marshal(mxConfigData)
	if err != nil {
		return err
	}

	// Add MX_CONFIG environment variable.
	for i := range podTemplate.Spec.Containers {

		c := &podTemplate.Spec.Containers[i]

		// Set environment variable MX_CONFIG
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  mxConfig,
			Value: string(mxConfigJson),
		})

		// Set Mxnet Distributed Training environment variable
		// We get these envs from MX_COFING to make them stay identical
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_PS_ROOT_PORT",
			Value: strconv.Itoa(getConfigAddr(&mxConfigData, mxv1.MXReplicaTypeScheduler, 0).Port),
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_PS_ROOT_URI",
			Value: getConfigAddr(&mxConfigData, mxv1.MXReplicaTypeScheduler, 0).Url,
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_NUM_SERVER",
			Value: strconv.Itoa(getConfigReplica(&mxConfigData, mxv1.MXReplicaTypeServer)),
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_NUM_WORKER",
			Value: strconv.Itoa(getConfigReplica(&mxConfigData, mxv1.MXReplicaTypeWorker)),
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_ROLE",
			Value: mxConfigData.Task.Type,
		})

		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_USE_KUBERNETES",
			Value: strconv.Itoa(1),
		})

		// BytePS needs env DMLC_WORKER_ID for each worker
		addBytePSEnv(c, rtype, index)
	}
	return nil
}

func setRestartPolicy(podTemplateSpec *corev1.PodTemplateSpec, spec *commonv1.ReplicaSpec) {
	if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = corev1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = corev1.RestartPolicy(spec.RestartPolicy)
	}
}

func getConfigAddr(mxConfigData *MXConfig, rtype commonv1.ReplicaType, index int) UrlPort {
	rt := strings.ToLower(string(rtype))
	var url_port UrlPort
	if len(mxConfigData.Cluster[rt]) <= index {
		// index out of range, maybe this url doen't exist
		url_port = UrlPort{
			Url:  "",
			Port: 0,
		}
	} else {
		url_port = mxConfigData.Cluster[rt][index]
	}
	return url_port
}

func getConfigReplica(mxConfigData *MXConfig, rtype commonv1.ReplicaType) int {
	rt := strings.ToLower(string(rtype))
	return len(mxConfigData.Cluster[rt])
}

func addBytePSEnv(c *corev1.Container, rtype, index string) {
	if rtype == strings.ToLower(string(mxv1.MXReplicaTypeWorker)) {
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "DMLC_WORKER_ID",
			Value: index,
		})
	}
}
