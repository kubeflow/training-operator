/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"testing"

	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-common/support"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kftov1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

func PytorchJob(t Test, namespace, name string) func(g gomega.Gomega) *kftov1.PyTorchJob {
	return func(g gomega.Gomega) *kftov1.PyTorchJob {
		job, err := t.Client().Kubeflow().KubeflowV1().PyTorchJobs(namespace).Get(t.Ctx(), name, metav1.GetOptions{})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return job
	}
}

func PytorchJobCondition(job *kftov1.PyTorchJob) string {
	if len(job.Status.Conditions) == 0 {
		return ""
	}
	return job.Status.Conditions[len(job.Status.Conditions)-1].Reason
}

func TestPytorchjobWithSFTtrainer(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// Create a namespace
	namespace := test.NewTestNamespace()
	config := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-config",
			Namespace: namespace.Name,
			Labels: map[string]string{
				"kueue.x-k8s.io/queue-name": "lq-trainer",
			},
		},
		BinaryData: map[string][]byte{
			"config.json":                   ReadFile(test, "config.json"),
			"twitter_complaints_small.json": ReadFile(test, "twitter_complaints_small.json"),
		},
		Immutable: Ptr(true),
	}

	config, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), config, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", config.Namespace, config.Name)

	tuningJob := &kftov1.PyTorchJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PyTorchJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kfto-sft",
			Namespace: namespace.Name,
		},
		Spec: kftov1.PyTorchJobSpec{
			PyTorchReplicaSpecs: map[kftov1.ReplicaType]*kftov1.ReplicaSpec{
				"Master": &kftov1.ReplicaSpec{
					Replicas:      Ptr(int32(1)),
					RestartPolicy: "Never",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "pytorch",
									Image:           "quay.io/tedchang/sft-trainer:dev",
									ImagePullPolicy: corev1.PullIfNotPresent,
									Command:         []string{"python", "/app/launch_training.py"},
									Env: []corev1.EnvVar{
										{
											Name:  "SFT_TRAINER_CONFIG_JSON_PATH",
											Value: "/etc/config/config.json",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "config-volume",
											MountPath: "/etc/config",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "config-volume",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "my-config",
											},
											Items: []corev1.KeyToPath{
												{
													Key:  "config.json",
													Path: "config.json",
												},
												{
													Key:  "twitter_complaints_small.json",
													Path: "twitter_complaints_small.json",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tuningJob, err = test.Client().Kubeflow().KubeflowV1().PyTorchJobs(namespace.Name).Create(test.Ctx(), tuningJob, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created PytorchJob %s/%s successfully", tuningJob.Namespace, tuningJob.Name)

	test.Eventually(PytorchJob(test, namespace.Name, tuningJob.Name), TestTimeoutLong).Should(WithTransform(PytorchJobCondition, Equal("PyTorchJobSucceeded")))
	test.T().Logf("PytorchJob %s/%s ran successfully", tuningJob.Namespace, tuningJob.Name)
}
