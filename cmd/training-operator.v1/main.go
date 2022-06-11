/*
Copyright 2021.

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
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	commonutil "github.com/kubeflow/common/pkg/util"
	trainingv1 "github.com/kubeflow/training-operator/pkg/apis/training/v1"
	"github.com/kubeflow/training-operator/pkg/config"
	controllerv1 "github.com/kubeflow/training-operator/pkg/controller.v1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(trainingv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enabledSchemes controllerv1.EnabledSchemes
	var enableGangScheduling bool
	var gangSchedulerName string
	var namespace string
	var monitoringPort int
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Var(&enabledSchemes, "enable-scheme", "Enable scheme(s) as --enable-scheme=tfjob --enable-scheme=pytorchjob, case insensitive."+
		" Now supporting TFJob, PyTorchJob, MXNetJob, XGBoostJob. By default, all supported schemes will be enabled.")
	flag.BoolVar(&enableGangScheduling, "enable-gang-scheduling", false, "Set true to enable gang scheduling")
	flag.StringVar(&gangSchedulerName, "gang-scheduler-name", "volcano", "The scheduler to gang-schedule kubeflow jobs, defaults to volcano")
	flag.StringVar(&namespace, "namespace", os.Getenv(commonutil.EnvKubeflowNamespace), "The namespace to monitor kubeflow jobs. If unset, it monitors all namespaces cluster-wide."+
		"If set, it only monitors kubeflow jobs in the given namespace.")
	flag.IntVar(&monitoringPort, "monitoring-port", 9443, "Endpoint port for displaying monitoring metrics. "+
		"It can be set to \"0\" to disable the metrics serving.")

	// PyTorch related flags
	flag.StringVar(&config.Config.PyTorchInitContainerImage, "pytorch-init-container-image",
		config.PyTorchInitContainerImageDefault, "The image for pytorch init container")
	flag.StringVar(&config.Config.PyTorchInitContainerTemplateFile, "pytorch-init-container-template-file",
		config.PyTorchInitContainerTemplateFileDefault, "The template file for pytorch init container")

	// MPI related flags
	flag.StringVar(&config.Config.MPIKubectlDeliveryImage, "mpi-kubectl-delivery-image",
		config.MPIKubectlDeliveryImageDefault, "The image for mpi launcher init container")

	opts := zap.Options{
		Development:     true,
		StacktraceLevel: zapcore.DPanicLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   monitoringPort,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "1ca428e5.",
		Namespace:              namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// TODO: We need a general manager. all rest reconciler addsToManager
	// Based on the user configuration, we start different controllers
	if enabledSchemes.Empty() {
		enabledSchemes.FillAll()
	}
	for _, s := range enabledSchemes {
		setupFunc, supported := controllerv1.SupportedSchemeReconciler[s]
		if !supported {
			setupLog.Error(fmt.Errorf("cannot find %s in supportedSchemeReconciler", s),
				"scheme not supported", "scheme", s)
			os.Exit(1)
		}
		if err = setupFunc(mgr, enableGangScheduling); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", s)
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
