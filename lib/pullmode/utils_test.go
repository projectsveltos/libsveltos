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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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

		labels := pullmode.GetConfigurationBundleLabels(clusterName, requestorKind, requestorName,
			requestorFeature, requestorIndex)

		Expect(pullmode.CreateConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, name, getResources(),
			labels, false, false, logger)).To(Succeed())

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

		labels := pullmode.GetConfigurationBundleLabels(clusterName, requestorKind, requestorName,
			requestorFeature, requestorIndex)

		Expect(pullmode.CreateConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, name, nil,
			labels, false, false, logger)).To(Succeed())

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
		Expect(pullmode.UpdateConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, name, resources,
			false, true, logger)).To(Succeed())

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

		name, err := pullmode.ReconcileConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, requestorIndex, nil, false, false, logger)
		Expect(err).To(BeNil())

		Eventually(func() error {
			currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
				currentConfigBundle)
			return err
		}, time.Minute, time.Second).Should(BeNil())

		resources := getResources()
		newName, err := pullmode.ReconcileConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, requestorIndex, resources, false, true, logger)
		Expect(err).To(BeNil())
		Expect(newName).To(Equal(name))

		Consistently(func() bool {
			currentConfigBundle := &libsveltosv1beta1.ConfigurationBundle{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
				currentConfigBundle)
			if err != nil {
				return false
			}
			return len(currentConfigBundle.Spec.Resources) == len(resources)
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("getStagedConfigurationBundles returns only non referenced ConfigurationBundles", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()
		requestorIndex := randomString()

		createNamespace(clusterNamespace)

		configurationGroup := &libsveltosv1beta1.ConfigurationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: clusterNamespace,
				Labels:    pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorName, requestorFeature),
			},
			Spec: libsveltosv1beta1.ConfigurationGroupSpec{
				ConfigurationItems: make([]libsveltosv1beta1.ConfigurationItem, 0),
			},
		}

		labels := pullmode.GetConfigurationBundleLabels(clusterName, requestorKind, requestorName,
			requestorFeature, requestorIndex)
		for i := 0; i < 3; i++ {
			bundle := createConfigurationBundle(clusterNamespace)
			bundle.Labels = labels
			Expect(k8sClient.Create(context.TODO(), bundle)).To(Succeed())

			configurationGroup.Spec.ConfigurationItems = append(configurationGroup.Spec.ConfigurationItems,
				libsveltosv1beta1.ConfigurationItem{
					ContentRef: &corev1.ObjectReference{
						Kind:       libsveltosv1beta1.ConfigurationBundleKind,
						APIVersion: libsveltosv1beta1.GroupVersion.String(),
						Name:       bundle.Name,
						Namespace:  clusterNamespace,
					},
				})
		}

		Expect(k8sClient.Create(context.TODO(), configurationGroup)).To(Succeed())

		stagedBundles := make(map[string]bool)
		for i := 0; i < 3; i++ {
			bundle := createConfigurationBundle(clusterNamespace)
			bundle.Labels = labels
			bundle.Labels[pullmode.StagedLabelKey] = pullmode.StagedLabelValue
			Expect(k8sClient.Create(context.TODO(), bundle)).To(Succeed())
			Expect(waitForObject(context.TODO(), k8sClient, bundle)).To(Succeed())
			stagedBundles[bundle.Name] = true
			time.Sleep(time.Second)
		}

		currentStagedBundles, err := pullmode.GetStagedConfigurationBundles(context.TODO(), k8sClient, clusterNamespace,
			clusterName, requestorKind, requestorName, requestorFeature, logger)
		Expect(err).To(BeNil())
		Expect(len(currentStagedBundles)).To(Equal(len(stagedBundles)))

		currentStagedBundleMap := make(map[string]bool, len(currentStagedBundles))
		for i := range currentStagedBundles {
			currentStagedBundleMap[currentStagedBundles[i].Name] = true
		}

		for k := range currentStagedBundleMap {
			_, ok := stagedBundles[k]
			Expect(ok).To(BeTrue())
		}

		// Verify bundles are sorted by creation time
		prevBundle := libsveltosv1beta1.ConfigurationBundle{}
		currentBundle := libsveltosv1beta1.ConfigurationBundle{}

		bundleName := currentStagedBundles[0].Name
		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: bundleName},
			&prevBundle)).To(Succeed())

		for i := 1; i < len(currentStagedBundles); i++ {
			bundleName = currentStagedBundles[i].Name
			Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: bundleName},
				&currentBundle)).To(Succeed())
			Expect(prevBundle.CreationTimestamp.Before(&currentBundle.CreationTimestamp)).To(BeTrue())
			prevBundle = currentBundle
		}
	})

	It("deleteStaleConfigurationBundles finds and deletes all current staged ConfigurationBundles", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()
		requestorIndex := randomString()

		createNamespace(clusterNamespace)

		referencedBundles := make([]pullmode.BundleData, 0)

		labels := pullmode.GetConfigurationBundleLabels(clusterName, requestorKind, requestorName,
			requestorFeature, requestorIndex)
		nonStagedNum := 5
		for i := 0; i < nonStagedNum; i++ {
			bundle := createConfigurationBundle(clusterNamespace)
			bundle.Labels = labels
			Expect(k8sClient.Create(context.TODO(), bundle)).To(Succeed())

			referencedBundles = append(referencedBundles, pullmode.BundleData{Name: bundle.Name})
		}

		stagedBundles := make(map[string]bool)
		for i := 0; i < 4; i++ {
			bundle := createConfigurationBundle(clusterNamespace)
			bundle.Labels = labels
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
			currentBundles := &libsveltosv1beta1.ConfigurationBundleList{}
			err := k8sClient.List(context.TODO(), currentBundles, listOptions...)
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

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorName, requestorFeature)

		name := randomString()

		action := libsveltosv1beta1.ActionDeploy
		bundles := getBundles()
		Expect(pullmode.CreateConfigurationGroup(context.TODO(), k8sClient, clusterNamespace, name, bundles,
			labels, action)).To(Succeed())

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

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorName, requestorFeature)

		action := libsveltosv1beta1.ActionDeploy
		bundles := getBundles()
		Expect(pullmode.CreateConfigurationGroup(context.TODO(), k8sClient, clusterNamespace, name, bundles,
			labels, action)).To(Succeed())

		Eventually(func() error {
			currentConfigGroup := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name},
				currentConfigGroup)
			return err
		}, time.Minute, time.Second).Should(BeNil())

		bundles = getBundles()
		Expect(pullmode.UpdateConfigurationGroup(context.TODO(), k8sClient, clusterNamespace, name, bundles,
			action, logger)).To(Succeed())

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

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorName, requestorFeature)

		// only one configurationGroup exists
		listOptions := []client.ListOption{
			client.InNamespace(clusterNamespace),
			labels,
		}

		Eventually(func() bool {
			configurationGroups := &libsveltosv1beta1.ConfigurationGroupList{}
			err := k8sClient.List(context.TODO(), configurationGroups, listOptions...)
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

func createConfigurationBundle(namespace string) *libsveltosv1beta1.ConfigurationBundle {
	return &libsveltosv1beta1.ConfigurationBundle{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      randomString(),
		},
	}
}
