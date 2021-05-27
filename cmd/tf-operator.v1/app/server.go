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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apiextensionclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	election "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	"github.com/kubeflow/common/pkg/util/signals"
	"github.com/kubeflow/tf-operator/cmd/tf-operator.v1/app/options"
	v1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	tfjobclientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/client/clientset/versioned/scheme"
	tfjobinformers "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	controller "github.com/kubeflow/tf-operator/pkg/controller.v1/tensorflow"
	"github.com/kubeflow/tf-operator/pkg/version"
)

const (
	apiVersion = "v1"
)

var (
	// leader election config
	leaseDuration = 15 * time.Second
	renewDuration = 5 * time.Second
	retryPeriod   = 3 * time.Second
)

// RecommendedKubeConfigPathEnv is the environment variable name for kubeconfig.
const RecommendedKubeConfigPathEnv = "KUBECONFIG"

var (
	isLeader = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tf_operator_is_leader",
		Help: "Is this client the leader of this tf-operator client set?",
	})
)

// Run runs the server.
func Run(opt *options.ServerOption) error {
	// Check if the -version flag was passed and, if so, print the version and exit.
	if opt.PrintVersion {
		version.PrintVersionAndExit(apiVersion)
	}

	namespace := os.Getenv(v1.EnvKubeflowNamespace)
	if len(namespace) == 0 {
		log.Infof("EnvKubeflowNamespace not set, use default namespace %s",
			metav1.NamespaceDefault)
		namespace = metav1.NamespaceDefault
	}
	if opt.Namespace == corev1.NamespaceAll {
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

	// Set client qps and burst by opt.
	kcfg.QPS = float32(opt.QPS)
	kcfg.Burst = opt.Burst
	log.Infof(
		"Creating client sets and informers with QPS %d, burst %d, resync period %s",
		opt.QPS, opt.Burst, opt.ResyncPeriod.String())

	// Create clients.
	kubeClientSet, leaderElectionClientSet,
		apiextensionClientSet, tfJobClientSet,
		volcanoClientSet, err := createClientSets(kcfg)
	if err != nil {
		log.Fatalf("Error create client set : %s", err.Error())
		return err
	}
	if !checkCRDExists(apiextensionClientSet, opt.Namespace) {
		return fmt.Errorf("Failed to get the expected TFJobs with API version %s",
			tfJobClientSet.KubeflowV1().RESTClient().APIVersion())
	}
	// Create informer factory.
	kubeInformerFactory := kubeinformers.NewFilteredSharedInformerFactory(kubeClientSet, opt.ResyncPeriod, opt.Namespace, nil)
	tfJobInformerFactory := tfjobinformers.NewSharedInformerFactory(tfJobClientSet, opt.ResyncPeriod)

	unstructuredInformer := controller.NewUnstructuredTFJobInformer(
		kcfg, opt.Namespace, opt.ResyncPeriod)

	// Create tf controller.
	tc := controller.NewTFController(unstructuredInformer, kubeClientSet, volcanoClientSet, tfJobClientSet, kubeInformerFactory, tfJobInformerFactory, *opt)

	// Start informer goroutines.
	go kubeInformerFactory.Start(stopCh)

	// We do not use the generated informer because of
	// https://github.com/kubeflow/tf-operator/issues/561
	// go tfJobInformerFactory.Start(stopCh)
	go unstructuredInformer.Informer().Run(stopCh)

	// Set leader election start function.
	run := func(context.Context) {
		isLeader.Set(1)
		if err := tc.Run(opt.Threadiness, stopCh); err != nil {
			log.Errorf("Failed to run the controller: %v", err)
		}
	}

	id, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %v", err)
	}
	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id = id + "_" + string(uuid.NewUUID())

	// Prepare event clients.
	eventBroadcaster := record.NewBroadcaster()
	if err = corev1.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("CoreV1 Add Scheme failed: %v", err)
	}
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "tf-operator"})

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
	election.RunOrDie(context.TODO(), election.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				isLeader.Set(0)
				log.Fatalf("leader election lost")
			},
		},
	})

	return nil
}

func createClientSets(config *restclientset.Config) (
	kubeclientset.Interface, kubeclientset.Interface,
	apiextensionclientset.Interface, tfjobclientset.Interface,
	volcanoclient.Interface, error) {

	kubeClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "tf-operator"))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	leaderElectionClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "leader-election"))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	apiextensionClientSet, err := apiextensionclientset.NewForConfig(restclientset.AddUserAgent(config, "leader-election"))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	tfJobClientSet, err := tfjobclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	volcanoClientSet, err := volcanoclient.NewForConfig(restclientset.AddUserAgent(config, "volcano"))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return kubeClientSet, leaderElectionClientSet, apiextensionClientSet, tfJobClientSet, volcanoClientSet, nil
}

// checkCRDExists checks if the CRD exists.
func checkCRDExists(clientset apiextensionclientset.Interface, namespace string) bool {
	crd, err := clientset.ApiextensionsV1beta1().
		CustomResourceDefinitions().
		Get(v1.TFCRD, metav1.GetOptions{})

	if err != nil {
		log.Error(err)
		if _, ok := err.(*errors.StatusError); ok {
			if errors.IsNotFound(err) {
				return false
			}
		} else {
			return false
		}
	}

	log.Infof("CRD %s/%s %s is registered",
		crd.Spec.Group, crd.Spec.Version, crd.Spec.Names.Singular)
	return true
}
