// Copyright 2018 The Kubeflow Authors
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
// limitations under the License.

package app

import (
	"fmt"
	"os"
	"time"

	"github.com/kubeflow/tf-operator/cmd/tf-operator.v1beta1/app/options"
	"github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1beta1"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	controller "github.com/kubeflow/tf-operator/pkg/controller.v1beta1/tensorflow"
	"github.com/kubeflow/tf-operator/pkg/util/signals"
	"github.com/kubeflow/tf-operator/pkg/version"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	election "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

const (
	apiVersion = "v1beta1"
)

var (
	// leader election config
	leaseDuration = 15 * time.Second
	renewDuration = 5 * time.Second
	retryPeriod   = 3 * time.Second
	resyncPeriod  = 30 * time.Second
)

const RecommendedKubeConfigPathEnv = "KUBECONFIG"

func Run(opt *options.ServerOption) error {
	// Check if the -version flag was passed and, if so, print the version and exit.
	if opt.PrintVersion {
		version.PrintVersionAndExit(apiVersion)
	}

	namespace := os.Getenv(v1beta1.EnvKubeflowNamespace)
	if len(namespace) == 0 {
		log.Infof("EnvKubeflowNamespace not set, use default namespace")
		namespace = metav1.NamespaceDefault
	}
	if opt.Namespace == v1.NamespaceAll {
		log.Info("Using cluster scoped operator")
	} else {
		log.Infof("Scoping operator to namespace %s", opt.Namespace)
	}

	// To help debugging, immediately log version.
	log.Infof("%+v", version.Info(apiVersion))

	// Set up signals so we handle the first shutdown signal gracefully.
	stopCh := signals.SetupSignalHandler()

	// Note: ENV KUBECONFIG will overwrite user defined Kubeconfig option.
	if len(os.Getenv(RecommendedKubeConfigPathEnv)) > 0 {
		// use the current context in kubeconfig
		// This is very useful for running locally.
		opt.Kubeconfig = os.Getenv(RecommendedKubeConfigPathEnv)
	}

	// Get kubernetes config.
	kcfg, err := clientcmd.BuildConfigFromFlags(opt.MasterURL, opt.Kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	// Create clients.
	kubeClientSet, leaderElectionClientSet, tfJobClientSet, err := createClientSets(kcfg)
	if err != nil {
		return err
	}

	// Create informer factory.
	kubeInformerFactory := kubeinformers.NewFilteredSharedInformerFactory(kubeClientSet, resyncPeriod, opt.Namespace, nil)
	tfJobInformerFactory := tfjobinformers.NewSharedInformerFactory(tfJobClientSet, resyncPeriod)

	unstructuredInformer := controller.NewUnstructuredTFJobInformer(kcfg, opt.Namespace)

	// Create tf controller.
	tc := controller.NewTFController(unstructuredInformer, kubeClientSet, tfJobClientSet, kubeInformerFactory, tfJobInformerFactory, *opt)

	// Start informer goroutines.
	go kubeInformerFactory.Start(stopCh)

	// We do not use the generated informer because of
	// https://github.com/kubeflow/tf-operator/issues/561
	// go tfJobInformerFactory.Start(stopCh)
	go unstructuredInformer.Informer().Run(stopCh)

	// Set leader election start function.
	run := func(<-chan struct{}) {
		if err := tc.Run(opt.Threadiness, stopCh); err != nil {
			log.Errorf("Failed to run the controller: %v", err)
		}
	}

	id, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %v", err)
	}

	// Prepare event clients.
	eventBroadcaster := record.NewBroadcaster()
	if err = v1.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("CoreV1 Add Scheme failed: %v", err)
	}
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "tf-operator"})

	rl := &resourcelock.EndpointsLock{
		EndpointsMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "tf-operator",
		},
		Client: leaderElectionClientSet.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		},
	}

	// Start leader election.
	election.RunOrDie(election.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				log.Fatalf("leader election lost")
			},
		},
	})

	return nil
}

func createClientSets(config *restclientset.Config) (kubeclientset.Interface, kubeclientset.Interface, tfjobclientset.Interface, error) {

	crdClient, err := crdclient.NewForConfig(config)

	if err != nil {
		return nil, nil, nil, err
	}

	checkCRDExists(crdClient, v1beta1.TFCRD)

	kubeClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "tf-operator"))
	if err != nil {
		return nil, nil, nil, err
	}

	leaderElectionClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "leader-election"))
	if err != nil {
		return nil, nil, nil, err
	}

	tfJobClientSet, err := tfjobclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	return kubeClientSet, leaderElectionClientSet, tfJobClientSet, nil
}

func checkCRDExists(clientset crdclient.Interface, crdName string) {
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
