/*
Copyright 2019 kubeflow.org.

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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-openapi/spec"
	mxJob "github.com/kubeflow/tf-operator/pkg/apis/mxnet/v1"
	pytorchJob "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1"
	tfjob "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	xgboostJob "github.com/kubeflow/tf-operator/pkg/apis/xgboost/v1"
	"k8s.io/klog"
	"k8s.io/kube-openapi/pkg/common"
)

// Generate OpenAPI spec definitions for TFJob Resource
func main() {
	if len(os.Args) <= 2 {
		klog.Fatal("Supply a framework and version")
	}
	framework := os.Args[1]
	version := os.Args[2]
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	var oAPIDefs map[string]common.OpenAPIDefinition

	switch framework {
	case "tensorflow":
		oAPIDefs = tfjob.GetOpenAPIDefinitions(func(name string) spec.Ref {
			return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name, framework)))
		})
	case "pytorch":
		oAPIDefs = pytorchJob.GetOpenAPIDefinitions(func(name string) spec.Ref {
			return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name, framework)))
		})
	case "mxnet":
		oAPIDefs = mxJob.GetOpenAPIDefinitions(func(name string) spec.Ref {
			return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name, framework)))
		})
	case "xgboost":
		oAPIDefs = xgboostJob.GetOpenAPIDefinitions(func(name string) spec.Ref {
			return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name, framework)))
		})
	}

	defs := spec.Definitions{}
	for defName, val := range oAPIDefs {
		defs[swaggify(defName, framework)] = val.Schema
	}
	swagger := spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:     "2.0",
			Definitions: defs,
			Paths:       &spec.Paths{Paths: map[string]spec.PathItem{}},
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:       framework,
					Description: fmt.Sprintf("Python SDK for %v", framework),
					Version:     version,
				},
			},
		},
	}
	jsonBytes, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		klog.Fatal(err.Error())
	}
	fmt.Println(string(jsonBytes))
}

func swaggify(name, framework string) string {
	name = strings.Replace(name, fmt.Sprintf("github.com/kubeflow/tf-operator/pkg/apis/%s/", framework), "", -1)
	name = strings.Replace(name, "github.com/kubeflow/common/pkg/apis/common/", "", -1)
	name = strings.Replace(name, "/", ".", -1)
	return name
}
