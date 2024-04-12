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
// limitations under the License.

package mxnet

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/controller.v1/common"
	mxnetwebhook "github.com/kubeflow/training-operator/pkg/webhooks/mxnet"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	testK8sClient client.Client
	testEnv       *envtest.Environment
	testCtx       context.Context
	testCancel    context.CancelFunc
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testCtx, testCancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "manifests", "base", "crds")},
		ErrorIfCRDPathMissing: true,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "manifests", "base", "webhook", "manifests.yaml")},
		},
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = kubeflowv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	testK8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(testK8sClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		WebhookServer: webhook.NewServer(
			webhook.Options{
				Host:    testEnv.WebhookInstallOptions.LocalServingHost,
				Port:    testEnv.WebhookInstallOptions.LocalServingPort,
				CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
			}),
	})
	Expect(err).NotTo(HaveOccurred())

	gangSchedulingSetupFunc := common.GenNonGangSchedulerSetupFunc()
	r := NewReconciler(mgr, gangSchedulingSetupFunc)

	Expect(r.SetupWithManager(mgr, 1)).NotTo(HaveOccurred())
	Expect(mxnetwebhook.SetupWebhook(mgr)).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(testCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", testEnv.WebhookInstallOptions.LocalServingHost, testEnv.WebhookInstallOptions.LocalServingPort)
	Eventually(func(g Gomega) {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(conn.Close()).NotTo(HaveOccurred())
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	testCancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
