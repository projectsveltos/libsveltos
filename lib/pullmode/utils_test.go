/*
Copyright 2025. projectsveltos.io. All rights reserved.

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

package pullmode_test

import (
	"context"
	"maps"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
	"github.com/projectsveltos/libsveltos/lib/pullmode"
)

var _ = Describe("Utils for pullmode APIs", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig())
	})

	It("createConfigurationBundle creates ConfigurationBundle", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()
		requestorIndex := randomString()

		createNamespace(clusterNamespace)

		name := randomString()

		labels := pullmode.GetConfigurationBundleLabels(clusterName, requestorKind, requestorFeature)

		bundle, err := pullmode.CreateConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, name, requestorName,
			requestorIndex, getResources(), labels, false, false, logger)
		Expect(err).To(BeNil())
		Expect(bundle).ToNot(BeNil())

		Eventually(func() error {
			currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
				currentConfigBundle)
			return err
		}, time.Minute, time.Second).Should(BeNil())

		currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
			currentConfigBundle)).To(Succeed())
		Expect(currentConfigBundle.Labels).ToNot(BeNil())

		for k := range labels {
			v, ok := currentConfigBundle.Labels[k]
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(labels[k]))
		}

		Expect(currentConfigBundle.Spec.Resources).ToNot(BeNil())
		Expect(len(currentConfigBundle.Spec.Resources)).To(Equal(len(getResources())))
	})

	It("updateConfigurationBundle updates ConfigurationBundle", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()
		requestorIndex := randomString()

		createNamespace(clusterNamespace)

		name := randomString()

		labels := pullmode.GetConfigurationBundleLabels(clusterName, requestorKind, requestorFeature)

		bundle, err := pullmode.CreateConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, name, requestorName,
			requestorIndex, nil, labels, false, false, logger)
		Expect(err).To(BeNil())
		Expect(bundle).ToNot(BeNil())

		Eventually(func() error {
			currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
				currentConfigBundle)
			return err
		}, time.Minute, time.Second).Should(BeNil())

		currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
			currentConfigBundle)).To(Succeed())
		Expect(currentConfigBundle.Labels).ToNot(BeNil())

		Expect(currentConfigBundle.Spec.Resources).To(BeNil())

		resources := getResources()
		bundle, err = pullmode.UpdateConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, name, requestorName,
			requestorIndex, resources, false, true, logger)
		Expect(err).To(BeNil())
		Expect(bundle).ToNot(BeNil())

		Eventually(func() bool {
			currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
				currentConfigBundle)
			if err != nil {
				return false
			}
			return len(currentConfigBundle.Spec.Resources) == len(resources)
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("reconcileConfigurationBundle creates/updates one ConfigurationBundle for same requestor", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()
		requestorIndex := randomString()

		createNamespace(clusterNamespace)

		bundle, err := pullmode.ReconcileConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, requestorIndex, nil, false, false, logger)
		Expect(err).To(BeNil())

		Eventually(func() error {
			currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: bundle.Name},
				currentConfigBundle)
			return err
		}, time.Minute, time.Second).Should(BeNil())

		resources := getResources()
		newBundle, err := pullmode.ReconcileConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, requestorIndex, resources, false, true, logger)
		Expect(err).To(BeNil())
		Expect(newBundle).ToNot(BeNil())
		Expect(newBundle.Name).To(Equal(bundle.Name))

		Consistently(func() bool {
			currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: bundle.Name},
				currentConfigBundle)
			if err != nil {
				return false
			}
			return len(currentConfigBundle.Spec.Resources) == len(resources)
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("deleteStaleConfigurationBundles finds and deletes all current staged ConfigurationBundles", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		referencedBundles := make([]pullmode.BundleData, 0)

		labels := pullmode.GetConfigurationBundleLabels(clusterName, requestorKind, requestorFeature)
		nonStagedNum := 5
		for range nonStagedNum {
			bundle := createConfigurationBundle(clusterNamespace, requestorName, labels)
			Expect(k8sClient.Create(context.TODO(), bundle)).To(Succeed())
			Expect(waitForObject(context.TODO(), k8sClient, bundle)).To(Succeed())
			referencedBundles = append(referencedBundles, pullmode.BundleData{Name: bundle.Name})
		}

		stagedBundles := make(map[string]bool)
		for range 4 {
			bundle := createConfigurationBundle(clusterNamespace, requestorName, labels)
			bundle.Labels[pullmode.StagedLabelKey] = pullmode.StagedLabelValue
			Expect(k8sClient.Create(context.TODO(), bundle)).To(Succeed())
			Expect(waitForObject(context.TODO(), k8sClient, bundle)).To(Succeed())
			stagedBundles[bundle.Name] = true
			time.Sleep(time.Second)
		}

		err := pullmode.DeleteStaleConfigurationBundles(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, referencedBundles, logger)
		Expect(err).To(BeNil())

		listOptions := []client.ListOption{
			client.InNamespace(clusterNamespace),
			labels,
		}
		// Verify all staged ConfigurationBundles are deleted
		Eventually(func() bool {
			currentBundles, err := pullmode.GetConfigurationBundles(context.TODO(), k8sClient, clusterNamespace,
				requestorName, "", labels)
			if err != nil {
				return false
			}
			return len(currentBundles.Items) == nonStagedNum
		}, time.Minute, time.Second).Should(BeTrue())

		// Verify all NON staged ConfigurationBundles are still present
		currentBundles := &libsveltosv1beta1.ConfigurationBundleList{}
		Expect(k8sClient.List(context.TODO(), currentBundles, listOptions...)).To(Succeed())
		Expect(len(currentBundles.Items)).To(Equal(nonStagedNum)) // number of non staged configurationBundles test created

	})

	It("createConfigurationGroup creates ConfigurationGroup", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		name := randomString()

		action := libsveltosv1beta1.ActionDeploy
		bundles := getBundles()
		Expect(pullmode.CreateConfigurationGroup(context.TODO(), k8sClient, clusterNamespace, name, requestorName,
			bundles, labels, action)).To(Succeed())

		Eventually(func() error {
			currentConfigGroup := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
				currentConfigGroup)
			return err
		}, time.Minute, time.Second).Should(BeNil())

		currentConfigGroup := &libsveltosv1beta1.ConfigurationGroup{}
		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
			currentConfigGroup)).To(Succeed())
		Expect(currentConfigGroup.Spec.Action).To(Equal(action))
		Expect(currentConfigGroup.Labels).ToNot(BeNil())

		for k := range labels {
			v, ok := currentConfigGroup.Labels[k]
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(labels[k]))
		}

		bundleMap := make(map[string][]byte, 0)
		for i := range bundles {
			bundleMap[bundles[i].Name] = bundles[i].Hash
		}

		Expect(currentConfigGroup.Spec.ConfigurationItems).ToNot(BeNil())
		Expect(len(currentConfigGroup.Spec.ConfigurationItems)).To(Equal(len(bundles)))
		for i := range currentConfigGroup.Spec.ConfigurationItems {
			item := &currentConfigGroup.Spec.ConfigurationItems[i]
			Expect(item.ContentRef).ToNot(BeNil())
			v, ok := bundleMap[item.ContentRef.Name]
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(item.Hash))
		}
		Expect(len(currentConfigGroup.Finalizers)).To(Equal(0))
	})

	It("updateConfigurationGroup updates ConfigurationGroup", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		name := randomString()

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		action := libsveltosv1beta1.ActionDeploy
		bundles := getBundles()
		Expect(pullmode.CreateConfigurationGroup(context.TODO(), k8sClient, clusterNamespace, name,
			requestorName, bundles, labels, action)).To(Succeed())

		Eventually(func() error {
			currentConfigGroup := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
				currentConfigGroup)
			return err
		}, time.Minute, time.Second).Should(BeNil())

		bundles = getBundles()
		Expect(pullmode.UpdateConfigurationGroup(context.TODO(), k8sClient, clusterNamespace, name,
			bundles, action, logger)).To(Succeed())

		currentConfigGroup := &libsveltosv1beta1.ConfigurationGroup{}
		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
			currentConfigGroup)).To(Succeed())
		Expect(currentConfigGroup.Spec.Action).To(Equal(action))
		Expect(currentConfigGroup.Labels).ToNot(BeNil())

		bundleMap := make(map[string][]byte, 0)
		for i := range bundles {
			bundleMap[bundles[i].Name] = bundles[i].Hash
		}

		Expect(currentConfigGroup.Spec.ConfigurationItems).ToNot(BeNil())
		Expect(len(currentConfigGroup.Spec.ConfigurationItems)).To(Equal(len(bundles)))
		for i := range currentConfigGroup.Spec.ConfigurationItems {
			item := &currentConfigGroup.Spec.ConfigurationItems[i]
			Expect(item.ContentRef).ToNot(BeNil())
			v, ok := bundleMap[item.ContentRef.Name]
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(item.Hash))
		}
	})

	It("reconcileConfigurationGroup creates/updates one ConfigurationGroup for same requestor", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		bundles := getBundles()

		Expect(pullmode.ReconcileConfigurationGroup(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, bundles, logger)).To(Succeed())

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		// only one configurationGroup exists
		listOptions := []client.ListOption{
			client.InNamespace(clusterNamespace),
			labels,
		}

		Eventually(func() bool {
			configurationGroups, err := pullmode.GetConfigurationGroups(context.TODO(), k8sClient,
				clusterNamespace, requestorName, labels)
			if err != nil {
				return false
			}
			return len(configurationGroups.Items) == 1
		}, time.Minute, time.Second).Should(BeTrue())

		bundles = getBundles()
		Expect(pullmode.ReconcileConfigurationGroup(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, bundles, logger)).To(Succeed())

		Consistently(func() bool {
			configurationGroups := &libsveltosv1beta1.ConfigurationGroupList{}
			err := k8sClient.List(context.TODO(), configurationGroups, listOptions...)
			if err != nil {
				return false
			}
			return len(configurationGroups.Items) == 1
		}, time.Minute, time.Second).Should(BeTrue())
	})
})

func getResources() []unstructured.Unstructured {
	namespace := `  apiVersion: v1
  kind: Namespace
  metadata:
    name: example-namespace
    labels:
      environment: development`

	serviceAccount := `  apiVersion: v1
  kind: ServiceAccount
  metadata:
    name: example-service-account
    namespace: example-namespace
    labels:
      app: example-app`

	role := `  apiVersion: rbac.authorization.k8s.io/v1
  kind: Role
  metadata:
    name: example-role
    namespace: example-namespace
  rules:
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "update", "patch"]
  - apiGroups: ["batch"]
    resources: ["jobs", "cronjobs"]
    verbs: ["get", "list", "create", "delete"]`

	roleBinding := `  apiVersion: rbac.authorization.k8s.io/v1
  kind: RoleBinding
  metadata:
    name: example-role-binding
    namespace: example-namespace
  subjects:
  - kind: ServiceAccount
    name: example-service-account
    namespace: example-namespace
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: Role
    name: example-role`

	resources := make([]unstructured.Unstructured, 0)
	uNamespace, err := k8s_utils.GetUnstructured([]byte(namespace))
	Expect(err).To(BeNil())
	resources = append(resources, *uNamespace)

	uServiceAccount, err := k8s_utils.GetUnstructured([]byte(serviceAccount))
	Expect(err).To(BeNil())
	resources = append(resources, *uServiceAccount)

	uRole, err := k8s_utils.GetUnstructured([]byte(role))
	Expect(err).To(BeNil())
	resources = append(resources, *uRole)

	uRoleBinding, err := k8s_utils.GetUnstructured([]byte(roleBinding))
	Expect(err).To(BeNil())
	resources = append(resources, *uRoleBinding)

	return resources
}

func getBundles() []pullmode.BundleData {
	bundles := make([]pullmode.BundleData, 0)

	for i := 0; i < 5; i++ {
		bundles = append(bundles, pullmode.BundleData{Name: randomString(), Hash: []byte(randomString())})
	}
	return bundles
}

func getValidations() []libsveltosv1beta1.ValidateHealth {
	validations := make([]libsveltosv1beta1.ValidateHealth, 0)
	for i := 0; i < 3; i++ {
		validations = append(validations, libsveltosv1beta1.ValidateHealth{
			FeatureID: libsveltosv1beta1.FeatureHelm,
			Group:     randomString(),
			Kind:      randomString(),
			Version:   randomString(),
			Name:      randomString(),
			Script:    randomString(),
		})
	}

	return validations
}

func createConfigurationBundle(namespace, requestorName string, lbls client.MatchingLabels) *libsveltosv1beta1.ConfigurationBundle {
	return &libsveltosv1beta1.ConfigurationBundle{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      randomString(),
			Labels:    maps.Clone(lbls),
			Annotations: map[string]string{
				pullmode.RequestorNameAnnotationKey: requestorName,
			},
		},
	}
}
