/*
Copyright 2022. projectsveltos.io. All rights reserved.

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

package utils_test

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2/textlogger"

	"github.com/projectsveltos/libsveltos/lib/utils"
)

const (
	viewClusterRole = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %s
rules:
- apiGroups: [""] # "" indicates the core API group
  resources: ["pods"]
  verbs: ["get", "watch", "list"]`

	serverValue = "https://127.0.0.1:63159"
	tokenID     = "token"
	caData      = "caData"
)

var _ = Describe("utils ", func() {
	It("GetUnstructured returns proper object", func() {
		policy, err := utils.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, randomString())))
		Expect(err).To(BeNil())
		Expect(policy.GetKind()).To(Equal("ClusterRole"))
	})

	It("GetDynamicResourceInterface returns dynamic resource interface", func() {
		policy, err := utils.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, randomString())))
		Expect(err).To(BeNil())

		dr, err := utils.GetDynamicResourceInterface(testEnv.Config, policy.GroupVersionKind(), policy.GetNamespace())
		Expect(err).To(BeNil())

		_, err = dr.Create(ctx, policy, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		const timeout = 20 * time.Second
		// Eventual loop so testEnv Cache is synced
		Eventually(func() error {
			currentClusterRole := &rbacv1.ClusterRole{}
			return testEnv.Get(context.TODO(),
				types.NamespacedName{Name: policy.GetName()}, currentClusterRole)
		}, timeout, time.Second).Should(BeNil())
	})

	It("should validate UserID and IDToken input in GetKubeconfigWithUserToken()", func() {
		_, err := utils.GetKubeconfigWithUserToken(ctx, nil, []byte(caData), "user@example.org", serverValue)
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("userID and IDToken cannot be empty"))

		_, err = utils.GetKubeconfigWithUserToken(ctx, []byte("0x123456"), []byte(caData), "", serverValue)
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("userID and IDToken cannot be empty"))
	})

	It("GetKubeconfigWithUserToken() should return a usable kubeconfig", func() {
		server := serverValue

		By("Calling GetKubeconfigWithUserToken()")
		kubeconfig, err := utils.GetKubeconfigWithUserToken(ctx, []byte("0x123456"), []byte(caData), "user@example.org", server)
		Expect(err).To(Succeed())
		Expect(kubeconfig).ToNot(BeNil())

		By("Checking that kubeconfig can be loaded")
		tmpFile, err := os.CreateTemp("", "kubeconfig")
		Expect(err).To(Succeed())
		defer tmpFile.Close()

		_, err = tmpFile.Write(kubeconfig)
		Expect(err).To(Succeed())

		_, err = clientcmd.BuildConfigFromFlags("", tmpFile.Name())
		Expect(err).To(Succeed())
	})

	It("GetKubernetesVersion returns cluster Kubernetes version", func() {
		version, err := utils.GetKubernetesVersion(context.TODO(), testEnv.Config,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(version).ToNot(BeEmpty())
	})
})
