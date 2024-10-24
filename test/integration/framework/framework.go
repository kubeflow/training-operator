/*
Copyright 2024 The Kubeflow Authors.

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

package framework

import (
	"context"
	"crypto/tls"
	"fmt"
	controllerv2 "github.com/kubeflow/training-operator/pkg/controller.v2"
	"net"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	schedulerpluginsv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"

	kubeflowv2 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v2alpha1"
	runtimecore "github.com/kubeflow/training-operator/pkg/runtime.v2/core"
	webhookv2 "github.com/kubeflow/training-operator/pkg/webhook.v2"
)

type Framework struct {
	testEnv *envtest.Environment
	cancel  context.CancelFunc
}

func (f *Framework) Init() *rest.Config {
	log.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))
	ginkgo.By("bootstrapping test environment")
	f.testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "manifests", "v2", "base", "crds"),
			filepath.Join("..", "..", "..", "manifests", "external-crds", "scheduler-plugins", "crd.yaml"),
			filepath.Join("..", "..", "..", "manifests", "external-crds", "jobset-operator"),
		},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "manifests", "v2", "base", "webhook", "manifests.yaml")},
		},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := f.testEnv.Start()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cfg).NotTo(gomega.BeNil())
	return cfg
}

func (f *Framework) RunManager(cfg *rest.Config, startControllers bool) (context.Context, client.Client) {
	webhookInstallOpts := &f.testEnv.WebhookInstallOptions
	gomega.ExpectWithOffset(1, kubeflowv2.AddToScheme(scheme.Scheme)).NotTo(gomega.HaveOccurred())
	gomega.ExpectWithOffset(1, jobsetv1alpha2.AddToScheme(scheme.Scheme)).NotTo(gomega.HaveOccurred())
	gomega.ExpectWithOffset(1, schedulerpluginsv1alpha1.AddToScheme(scheme.Scheme)).NotTo(gomega.HaveOccurred())

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())
	gomega.ExpectWithOffset(1, k8sClient).NotTo(gomega.BeNil())

	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	mgr, err := ctrl.NewManager(cfg, manager.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0", // disable metrics to avoid conflicts between packages.
		},
		WebhookServer: webhook.NewServer(
			webhook.Options{
				Host:    webhookInstallOpts.LocalServingHost,
				Port:    webhookInstallOpts.LocalServingPort,
				CertDir: webhookInstallOpts.LocalServingCertDir,
			}),
	})
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred(), "failed to create manager")

	runtimes, err := runtimecore.New(ctx, mgr.GetClient(), mgr.GetFieldIndexer())
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())
	gomega.ExpectWithOffset(1, runtimes).NotTo(gomega.BeNil())

	if startControllers {
		failedCtrlName, err := controllerv2.SetupControllers(mgr, runtimes)
		gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred(), "controller", failedCtrlName)
		gomega.ExpectWithOffset(1, failedCtrlName).To(gomega.BeEmpty())
	}

	failedWebhookName, err := webhookv2.Setup(mgr, runtimes)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred(), "webhook", failedWebhookName)
	gomega.ExpectWithOffset(1, failedWebhookName).To(gomega.BeEmpty())

	go func() {
		defer ginkgo.GinkgoRecover()
		err = mgr.Start(ctx)
		gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred(), "failed to run manager")
	}()

	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOpts.LocalServingHost, webhookInstallOpts.LocalServingPort)
	gomega.Eventually(func(g gomega.Gomega) {
		var conn *tls.Conn
		conn, err = tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(conn.Close()).Should(gomega.Succeed())
	}).Should(gomega.Succeed())
	return ctx, k8sClient
}

func (f *Framework) Teardown() {
	ginkgo.By("tearing down the test environment")
	if f.cancel != nil {
		f.cancel()
	}
	gomega.ExpectWithOffset(1, f.testEnv.Stop()).NotTo(gomega.HaveOccurred())
}
