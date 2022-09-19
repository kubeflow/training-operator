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

package tensorflow

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	v1beta1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	testK8sClient client.Client
	testEnv       *envtest.Environment
	testCtx       context.Context
	testCancel    context.CancelFunc
	reconciler    *TFJobReconciler
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	const (
		timeout  = 10 * time.Second
		interval = 1000 * time.Millisecond
	)
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testCtx, testCancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "manifests", "base", "crds")},
		ErrorIfCRDPathMissing: true,
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
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	reconciler = NewReconciler(mgr, false)
	Expect(reconciler.SetupWithManager(mgr)).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(testCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// This step is introduced to make sure cache starts before running any tests
	Eventually(func() error {
		nsList := &corev1.NamespaceList{}
		if err := testK8sClient.List(context.Background(), nsList); err != nil {
			return err
		} else if len(nsList.Items) < 1 {
			return fmt.Errorf("cannot get at lease one namespace, got %d", len(nsList.Items))
		}
		return nil
	}, timeout, interval).Should(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	testCancel()
	// Give 5 seconds to stop all tests
	time.Sleep(5 * time.Second)
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
