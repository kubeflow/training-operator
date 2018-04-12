// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	v1alpha1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha1"
	tfjobclient "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type opts struct {
	kubeConfigPath string
	useGPU         bool
	nrTFJobs       uint
	namespace      string
	schedulerName  string
}

func (o *opts) addFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.kubeConfigPath, "kube-config-path", "", "a path of k8s config")
	fs.UintVar(&o.nrTFJobs, "nr-tfjobs", 1, "a number of generated TFJobs")
	fs.BoolVar(&o.useGPU, "use-gpu", false, "generate TFJob which uses GPU")
	fs.StringVar(&o.namespace, "namespace", "default", "namespace of k8s")
	fs.StringVar(&o.schedulerName, "scheduler-name", "default", "scheduler name of k8s")
}

func tfjobTemplate(jobName string, gpu bool, schedulerName string) *v1alpha1.TFJob {
	one := int32(1)
	job := &v1alpha1.TFJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v1alpha1",
			Kind:       "TFJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Spec: v1alpha1.TFJobSpec{
			SchedulerName: schedulerName,
			ReplicaSpecs: []*v1alpha1.TFReplicaSpec{
				{
					Replicas: &one,
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "tensorflow",
								},
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}

	if !gpu {
		job.Spec.ReplicaSpecs[0].TFReplicaType = v1alpha1.WORKER
		job.Spec.ReplicaSpecs[0].Template.Spec.Containers[0].Image = "gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff"
	} else {
		job.Spec.ReplicaSpecs[0].TFReplicaType = v1alpha1.MASTER
		job.Spec.ReplicaSpecs[0].Template.Spec.Containers[0].Image = "gcr.io/tf-on-k8s-dogfood/tf_sample_gpu:dc944ff"
		job.Spec.TerminationPolicy = &v1alpha1.TerminationPolicySpec{
			Chief: &v1alpha1.ChiefSpec{
				ReplicaName: string(v1alpha1.MASTER),
			},
		}
	}

	return job
}

func main() {
	o := &opts{}
	o.addFlags(flag.CommandLine)
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", o.kubeConfigPath)
	if err != nil {
		fmt.Printf("failed to build config with kubeconfig %s: %v\n", o.kubeConfigPath, err)
		os.Exit(1)
	}

	tfJobClient, err := tfjobclient.NewForConfig(config)
	if err != nil {
		fmt.Printf("failed to create a tfjobclient %v", err)
		os.Exit(1)
	}

	timestamp := time.Now().Nanosecond()

	for i := uint(0); i < o.nrTFJobs; i++ {
		jobName := fmt.Sprintf("tfjob-%d-%d", timestamp, i)
		job := tfjobTemplate(jobName, o.useGPU, o.schedulerName)
		_, err := tfJobClient.KubeflowV1alpha1().TFJobs(o.namespace).Create(job)
		if err != nil {
			fmt.Printf("failed to create TFJob %s: %v\n", job.ObjectMeta.Name, err)
			os.Exit(1)
		}
	}
}
