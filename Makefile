# Image URL to use all building/pushing image targets
IMG ?= kubeflow/training-operator:latest
# CRD generation options
CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true,maxDescLen=400"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=training-operator webhook paths="./pkg/..." output:crd:artifacts:config=manifests/base/crds output:rbac:artifacts:config=manifests/base/rbac

generate: controller-gen ## Generate apidoc, sdk and code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate/boilerplate.go.txt" paths="./pkg/apis/..."
	hack/update-codegen.sh
	hack/python-sdk/gen-sdk.sh
	$(MAKE) apidoc

apidoc:
	hack/generate-apidoc.sh

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

GOLANGCI_LINT=$(shell which golangci-lint)
golangci-lint:
ifeq ($(GOLANGCI_LINT),)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.59.1
	$(info golangci-lint has been installed)
endif
	golangci-lint run --timeout 5m --go 1.21 ./...

ENVTEST_K8S_VERSION ?= 1.29
HAS_SETUP_ENVTEST := $(shell command -v setup-envtest;)

testall: manifests generate fmt vet golangci-lint test ## Run tests.

test: envtest
	KUBEBUILDER_ASSETS="$(shell setup-envtest use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

envtest:
ifndef HAS_SETUP_ENVTEST
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@bf15e44028f908c790721fc8fe67c7bf2d06a611 # v0.17.2
	@echo "setup-envtest has been installed"
endif
	@echo "setup-envtest has already installed"

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/training-operator.v1/main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/training-operator.v1/main.go

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} -f build/images/training-operator/Dockerfile .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build manifests/base/crds | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build manifests/base/crds | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd manifests/overlays/standalone && $(KUSTOMIZE) edit set image kubeflow/training-operator=${IMG}
	$(KUSTOMIZE) build manifests/overlays/standalone | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build manifests/overlays/standalone | kubectl delete -f -

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	GOBIN=$(PROJECT_DIR)/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	GOBIN=$(PROJECT_DIR)/bin go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.7
