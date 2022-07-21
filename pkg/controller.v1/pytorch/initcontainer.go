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
// limitations under the License

package pytorch

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/config"
)

var (
	initContainerTemplate = `
- name: init-pytorch
  image: {{.InitContainerImage}}
  imagePullPolicy: IfNotPresent
  resources:
    limits:
      cpu: 100m
      memory: 20Mi
    requests:
      cpu: 50m
      memory: 10Mi
  command: ['sh', '-c', 'until nslookup {{.MasterAddr}}; do echo waiting for master; sleep 2; done;']`
	onceInitContainer sync.Once
	icGenerator       *initContainerGenerator
)

type initContainerGenerator struct {
	template string
	image    string
}

func getInitContainerGenerator() *initContainerGenerator {
	onceInitContainer.Do(func() {
		icGenerator = &initContainerGenerator{
			template: getInitContainerTemplateOrDefault(config.Config.PyTorchInitContainerTemplateFile),
			image:    config.Config.PyTorchInitContainerImage,
		}
	})
	return icGenerator
}

func (i *initContainerGenerator) GetInitContainer(masterAddr string) ([]corev1.Container, error) {
	var buf bytes.Buffer
	tpl, err := template.New("container").Parse(i.template)
	if err != nil {
		return nil, err
	}
	if err := tpl.Execute(&buf, struct {
		MasterAddr         string
		InitContainerImage string
	}{
		MasterAddr:         masterAddr,
		InitContainerImage: i.image,
	}); err != nil {
		return nil, err
	}

	var result []corev1.Container
	err = yaml.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// getInitContainerTemplateOrDefault returns the init container template file if
// it exists, or return initContainerTemplate by default.
func getInitContainerTemplateOrDefault(file string) string {
	b, err := ioutil.ReadFile(file)
	if err == nil {
		return string(b)
	}
	return initContainerTemplate
}

func setInitContainer(obj interface{}, podTemplate *corev1.PodTemplateSpec,
	rtype, index string, log logr.Logger) error {
	pytorchJob, ok := obj.(*kubeflowv1.PyTorchJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of PyTorchJob", obj)
	}
	logger := log.WithValues(kubeflowv1.PytorchJobSingular, types.NamespacedName{
		Namespace: pytorchJob.Namespace,
		Name:      pytorchJob.Name,
	})

	// There is no need to set init container if no master is specified.
	if pytorchJob.Spec.PyTorchReplicaSpecs[kubeflowv1.PyTorchJobReplicaTypeMaster] == nil {
		logger.V(1).Info("No master is specified, skip setting init container")
		return nil
	}

	// Set the init container only if the master is specified and the current
	// rtype is worker.
	if rtype == strings.ToLower(string(kubeflowv1.PyTorchJobReplicaTypeWorker)) {
		g := getInitContainerGenerator()
		initContainers, err := g.GetInitContainer(genGeneralName(pytorchJob.Name,
			strings.ToLower(string(kubeflowv1.PyTorchJobReplicaTypeMaster)), strconv.Itoa(0)))
		if err != nil {
			return err
		}
		podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers,
			initContainers...)

	}
	return nil
}
