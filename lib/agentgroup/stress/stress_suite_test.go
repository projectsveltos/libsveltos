/*
Copyright 2026. projectsveltos.io. All rights reserved.

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

// Package stress_test runs the agentgroup Assigner against a real API server (envtest) with
// thousands of clusters. It is skipped unless AGENTGROUP_STRESS is set:
//
//	AGENTGROUP_STRESS=1 KUBEBUILDER_ASSETS=... go test ./lib/agentgroup/stress/ -v
//
// AGENTGROUP_STRESS_CLUSTERS, AGENTGROUP_STRESS_NAMESPACES, AGENTGROUP_STRESS_WORKERS and
// AGENTGROUP_STRESS_CAPACITY change the size (default 2000, 20, 50 and 5).
package stress_test

import (
	"context"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	agentNamespace = "stress-agents"

	// the API server of envtest is not throttled by the client
	restQPS   = 2000
	restBurst = 4000
)

var (
	testEnv    *envtest.Environment
	restConfig *rest.Config
	scheme     *runtime.Scheme
)

func TestStress(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AgentGroup Stress Suite")
}

var _ = BeforeSuite(func() {
	if os.Getenv("AGENTGROUP_STRESS") == "" {
		Skip("set AGENTGROUP_STRESS to run the stress test")
	}

	scheme = runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())

	testEnv = &envtest.Environment{}
	var err error
	restConfig, err = testEnv.Start()
	Expect(err).To(BeNil())
	restConfig.QPS = restQPS
	restConfig.Burst = restBurst

	c := newClient(nil)
	Expect(c.Create(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: agentNamespace}})).To(Succeed())
})

var _ = AfterSuite(func() {
	if testEnv != nil {
		Expect(testEnv.Stop()).To(Succeed())
	}
})

func envInt(name string, defaultValue int) int {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	Expect(err).To(BeNil())
	return n
}
