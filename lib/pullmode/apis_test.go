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
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/pullmode"
)

var _ = Describe("APIs for SveltosCluster instances in pullmode", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig())
	})

	It("GetRequestorKind/Namespace/Name return requestor kind/namespace/name", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		configGroup := &libsveltosv1beta1.ConfigurationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels:    labels,
				Annotations: map[string]string{
					pullmode.RequestorNameAnnotationKey: requestorName,
				},
			},
		}

		kind, err := pullmode.GetRequestorKind(configGroup)
		Expect(err).To(BeNil())
		Expect(kind).To(Equal(requestorKind))

		name, err := pullmode.GetRequestorName(configGroup)
		Expect(err).To(BeNil())
		Expect(name).To(Equal(requestorName))
	})

	It("RecordResourcesForDeployment creates ConfigurationGroup and ConfigurationBundles", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		resources := make(map[string][]unstructured.Unstructured)
		resources[randomString()] = getResources()

		setters := make([]pullmode.Option, 0)
		validations := getValidations()
		setters = append(setters, pullmode.WithValidateHealths(validations))
		Expect(pullmode.RecordResourcesForDeployment(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, resources, logger, setters...)).To(Succeed())

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
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

		configurationGroups := &libsveltosv1beta1.ConfigurationGroupList{}
		Expect(k8sClient.List(context.TODO(), configurationGroups, listOptions...)).To(Succeed())
		Expect(len(configurationGroups.Items[0].Spec.ConfigurationItems)).To(Equal(len(resources)))

		// Verify all configurationBundles referenced by ConfigurationGroup exists
		for i := range configurationGroups.Items[0].Spec.ConfigurationItems {
			item := &configurationGroups.Items[0].Spec.ConfigurationItems[i]

			configurationBundle := &libsveltosv1beta1.ConfigurationBundle{}
			Expect(k8sClient.Get(context.TODO(),
				types.NamespacedName{Namespace: clusterNamespace, Name: item.ContentRef.Name},
				configurationBundle)).To(Succeed())
		}

		// Copy current ConfigurationItems
		oldConfigurationItems := make([]libsveltosv1beta1.ConfigurationItem, len(configurationGroups.Items[0].Spec.ConfigurationItems))
		copy(oldConfigurationItems, configurationGroups.Items[0].Spec.ConfigurationItems)

		// Changes resources. Changing the key will cause a new ConfigurationBundle to be created.
		// One created previously will be removed
		resources = make(map[string][]unstructured.Unstructured)
		resources[randomString()] = getResources()
		Expect(pullmode.RecordResourcesForDeployment(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, resources, logger, setters...)).To(Succeed())

		// Wait for cache to sync. Make sure the ConfigurationGroup.Spec.ConfigurationItems has changed
		Eventually(func() bool {
			configurationGroups := &libsveltosv1beta1.ConfigurationGroupList{}
			err := k8sClient.List(context.TODO(), configurationGroups, listOptions...)
			if err != nil {
				return false
			}
			if len(configurationGroups.Items) != 1 {
				return false
			}
			return !reflect.DeepEqual(configurationGroups.Items[0].Spec.ConfigurationItems, oldConfigurationItems)
		}, time.Minute, time.Second).Should(BeTrue())

		// Get the update ConfigurationGroup
		Expect(k8sClient.List(context.TODO(), configurationGroups, listOptions...)).To(Succeed())

		// Verify the configurationBundle currently referenced exists
		for i := range configurationGroups.Items[0].Spec.ConfigurationItems {
			item := &configurationGroups.Items[0].Spec.ConfigurationItems[i]

			configurationBundle := &libsveltosv1beta1.ConfigurationBundle{}
			Expect(k8sClient.Get(context.TODO(),
				types.NamespacedName{Namespace: clusterNamespace, Name: item.ContentRef.Name},
				configurationBundle)).To(Succeed())
		}

		// Verify old ConfigurationBundles are removed
		configurationBundles := &libsveltosv1beta1.ConfigurationBundleList{}
		Expect(k8sClient.List(context.TODO(), configurationBundles, listOptions...)).To(Succeed())
		Expect(len(configurationBundles.Items)).To(Equal(len(resources)))
	})

	It("GetResourceDeploymentStatus return the resource deployment status", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		deploymentStatus := libsveltosv1beta1.FeatureStatusProvisioning
		failureMessage := randomString()
		lastAppliedTime := metav1.Time{Time: time.Now()}

		configurationGroup := &libsveltosv1beta1.ConfigurationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels:    labels,
				Annotations: map[string]string{
					pullmode.RequestorNameAnnotationKey: requestorName,
				},
			},
		}

		Expect(k8sClient.Create(context.TODO(), configurationGroup)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, configurationGroup)).To(Succeed())

		currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
			currentConfigurationGroup)).To(Succeed())

		currentConfigurationGroup.Status = libsveltosv1beta1.ConfigurationGroupStatus{
			DeploymentStatus: &deploymentStatus,
			FailureMessage:   &failureMessage,
			LastAppliedTime:  &lastAppliedTime,
		}
		Expect(k8sClient.Status().Update(context.TODO(), currentConfigurationGroup)).To(Succeed())

		// wait for cache to sync verifying status is actually set
		Eventually(func() bool {
			currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
				currentConfigurationGroup)
			if err != nil {
				return false
			}
			return currentConfigurationGroup.Status.DeploymentStatus != nil
		}, time.Minute, time.Second).Should(BeTrue())

		// Verify APIs returns the correct status
		status, err := pullmode.GetDeploymentStatus(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, logger)
		Expect(err).To(BeNil())

		Expect(status.DeploymentStatus).ToNot(BeNil())
		Expect(*status.DeploymentStatus).To(Equal(deploymentStatus))

		Expect(status.FailureMessage).ToNot(BeNil())
		Expect(*status.FailureMessage).To(Equal(failureMessage))

		Expect(status.LastAppliedTime).ToNot(BeNil())
	})

	It("StageResourcesForDeployment and CommitStagedResourcesForDeployment creates a ConfigurationGroup referencing stages bundles", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		stagedBundles := 3
		for range stagedBundles {
			requestorIndex := randomString()
			resources := make(map[string][]unstructured.Unstructured, 0)
			resources[requestorIndex] = getResources()
			Expect(pullmode.StageResourcesForDeployment(context.TODO(), k8sClient, clusterNamespace, clusterName,
				requestorKind, requestorName, requestorFeature, resources, false, logger)).To(Succeed())
		}

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		// wait for cache to sync
		Eventually(func() bool {
			currentConfigurationBundles, err := pullmode.GetConfigurationBundles(context.TODO(), k8sClient,
				clusterNamespace, requestorName, labels)
			if err != nil {
				return false
			}
			return len(currentConfigurationBundles.Items) == stagedBundles
		}, time.Minute, time.Second).Should(BeTrue())

		Expect(pullmode.CommitStagedResourcesForDeployment(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, logger)).To(Succeed())

		labels = pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)
		Eventually(func() bool {
			currentConfigurationGroups, err := pullmode.GetConfigurationGroups(context.TODO(), k8sClient,
				clusterNamespace, requestorName, labels)
			if err != nil {
				return false
			}
			return len(currentConfigurationGroups.Items) == 1
		}, time.Minute, time.Second).Should(BeTrue())

		currentConfigurationGroups, err := pullmode.GetConfigurationGroups(context.TODO(), k8sClient, clusterNamespace,
			requestorName, labels)
		Expect(err).To(BeNil())
		Expect(len(currentConfigurationGroups.Items)).To(Equal(1))
		Expect(len(currentConfigurationGroups.Items[0].Spec.ConfigurationItems)).To(Equal(stagedBundles))
		Expect(currentConfigurationGroups.Items[0].Spec.UpdatePhase).To(Equal(libsveltosv1beta1.UpdatePhaseReady))

		// Calling StageResourcesForDeployment set UpdatePhase to Preparing
		requestorIndex := randomString()
		resources := make(map[string][]unstructured.Unstructured, 0)
		resources[requestorIndex] = getResources()
		Expect(pullmode.StageResourcesForDeployment(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, resources, false, logger)).To(Succeed())

		Eventually(func() bool {
			currentConfigurationGroups, err := pullmode.GetConfigurationGroups(context.TODO(), k8sClient,
				clusterNamespace, requestorName, labels)
			if err != nil {
				return false
			}
			if len(currentConfigurationGroups.Items) != 1 {
				return false
			}

			return currentConfigurationGroups.Items[0].Spec.UpdatePhase == libsveltosv1beta1.UpdatePhasePreparing
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("RemoveResourcesFromDeployment marks a ConfigurationGroup for removal and removes all associated ConfigurationBundles", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		bundleNames := make([]string, 0)
		for i := 0; i < 3; i++ {
			bundle, err := pullmode.ReconcileConfigurationBundle(context.TODO(), k8sClient, clusterNamespace, clusterName,
				requestorKind, requestorName, requestorFeature, randomString(), nil, false, false, logger)
			Expect(err).To(BeNil())
			bundleNames = append(bundleNames, bundle.Name)
		}

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		configurationGroup := &libsveltosv1beta1.ConfigurationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels:    labels,
				Annotations: map[string]string{
					pullmode.RequestorNameAnnotationKey: requestorName,
				},
			},
		}

		configurationGroup.Spec.ConfigurationItems = make([]libsveltosv1beta1.ConfigurationItem, len(bundleNames))
		for i := range bundleNames {
			configurationGroup.Spec.ConfigurationItems[i] = libsveltosv1beta1.ConfigurationItem{
				ContentRef: &corev1.ObjectReference{
					Namespace:  clusterNamespace,
					Name:       bundleNames[i],
					Kind:       libsveltosv1beta1.ConfigurationBundleKind,
					APIVersion: libsveltosv1beta1.GroupVersion.String(),
				},
			}
		}
		configurationGroup.Spec.Action = libsveltosv1beta1.ActionDeploy
		Expect(k8sClient.Create(context.TODO(), configurationGroup)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, configurationGroup)).To(Succeed())

		Expect(pullmode.RemoveDeployedResources(context.TODO(), k8sClient, clusterNamespace, clusterName, requestorKind,
			requestorName, requestorFeature, logger)).To(BeNil())

		Eventually(func() bool {
			currentConfigurationGroups := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(),
				types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
				currentConfigurationGroups)
			if err != nil {
				return false
			}
			return currentConfigurationGroups.Spec.Action == libsveltosv1beta1.ActionRemove
		}, time.Minute, time.Second).Should(BeTrue())

		listOptions := []client.ListOption{
			client.InNamespace(clusterNamespace),
			labels,
		}

		// wait for cache to sync
		Eventually(func() bool {
			currentConfigurationBundles := &libsveltosv1beta1.ConfigurationBundleList{}
			err := k8sClient.List(context.TODO(), currentConfigurationBundles, listOptions...)
			if err != nil {
				return false
			}
			return len(currentConfigurationBundles.Items) == 0
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("GetResourceRemoveStatus return FeatureStatusRemoved when ConfigurationGroupd not present",
		func() {
			clusterNamespace := randomString()
			clusterName := randomString()
			requestorKind := randomString()
			requestorName := randomString()
			requestorFeature := randomString()

			createNamespace(clusterNamespace)

			status, err := pullmode.GetRemoveStatus(context.TODO(), k8sClient, clusterNamespace, clusterName,
				requestorKind, requestorName, requestorFeature, logger)
			Expect(err).ToNot(BeNil())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(status).To(BeNil())
		})

	It("TerminateDeploymentTracking deletes ConfigurationGroup", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		configurationGroup := &libsveltosv1beta1.ConfigurationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels:    labels,
				Annotations: map[string]string{
					pullmode.RequestorNameAnnotationKey: requestorName,
				},
			},
		}

		Expect(k8sClient.Create(context.TODO(), configurationGroup)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, configurationGroup)).To(Succeed())

		Expect(pullmode.TerminateDeploymentTracking(context.TODO(), k8sClient, clusterNamespace, clusterName, requestorKind,
			requestorName, requestorFeature, logger)).To(BeNil())

		Eventually(func() bool {
			currentConfigurationGroups := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(),
				types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
				currentConfigurationGroups)
			if err == nil {
				return false
			}
			return apierrors.IsNotFound(err) || !currentConfigurationGroups.DeletionTimestamp.IsZero()
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("IsBeingProvisioned returns true when content is being/has been deployed", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		requestorHash := []byte(randomString())
		configurationGroup := &libsveltosv1beta1.ConfigurationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels:    labels,
				Annotations: map[string]string{
					pullmode.RequestorNameAnnotationKey: requestorName,
				},
			},
			Spec: libsveltosv1beta1.ConfigurationGroupSpec{
				Action:        libsveltosv1beta1.ActionDeploy,
				RequestorHash: requestorHash,
				UpdatePhase:   libsveltosv1beta1.UpdatePhaseReady,
			},
		}

		Expect(k8sClient.Create(context.TODO(), configurationGroup)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, configurationGroup)).To(Succeed())

		currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
			currentConfigurationGroup)).To(Succeed())

		deploymentStatus := libsveltosv1beta1.FeatureStatusFailed
		failureMessage := randomString()
		lastAppliedTime := metav1.Time{Time: time.Now()}

		currentConfigurationGroup.Status = libsveltosv1beta1.ConfigurationGroupStatus{
			DeploymentStatus:      &deploymentStatus,
			FailureMessage:        &failureMessage,
			LastAppliedTime:       &lastAppliedTime,
			ObservedRequestorHash: requestorHash,
			ObservedGeneration:    currentConfigurationGroup.Generation,
		}
		Expect(k8sClient.Status().Update(context.TODO(), currentConfigurationGroup)).To(Succeed())

		// wait for cache to sync verifying status is actually set
		Eventually(func() bool {
			currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
				currentConfigurationGroup)
			if err != nil {
				return false
			}
			return currentConfigurationGroup.Status.DeploymentStatus != nil
		}, time.Minute, time.Second).Should(BeTrue())

		// Verify APIs returns the correct status
		isBeingProvisioned := pullmode.IsBeingProvisioned(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, logger)
		Expect(isBeingProvisioned).To(BeFalse())

		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
			currentConfigurationGroup)).To(Succeed())

		deploymentStatus = libsveltosv1beta1.FeatureStatusProvisioning
		currentConfigurationGroup.Status = libsveltosv1beta1.ConfigurationGroupStatus{
			DeploymentStatus:      &deploymentStatus,
			FailureMessage:        nil,
			LastAppliedTime:       &lastAppliedTime,
			ObservedRequestorHash: requestorHash,
			ObservedGeneration:    currentConfigurationGroup.Generation,
		}
		Expect(k8sClient.Status().Update(context.TODO(), currentConfigurationGroup)).To(Succeed())

		// wait for cache to sync verifying status is actually set
		Eventually(func() bool {
			currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
				currentConfigurationGroup)
			if err != nil {
				return false
			}
			return currentConfigurationGroup.Status.DeploymentStatus != nil &&
				*currentConfigurationGroup.Status.DeploymentStatus == deploymentStatus
		}, time.Minute, time.Second).Should(BeTrue())

		isBeingProvisioned = pullmode.IsBeingProvisioned(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, logger)
		Expect(isBeingProvisioned).To(BeTrue())
	})

	It("IsBeingRemoved returns true when applier is proceeding withdrawing deployed resources", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		requestorKind := randomString()
		requestorName := randomString()
		requestorFeature := randomString()

		createNamespace(clusterNamespace)

		labels := pullmode.GetConfigurationGroupLabels(clusterName, requestorKind, requestorFeature)

		requestorHash := []byte(randomString())
		configurationGroup := &libsveltosv1beta1.ConfigurationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      randomString(),
				Labels:    labels,
				Annotations: map[string]string{
					pullmode.RequestorNameAnnotationKey: requestorName,
				},
			},
			Spec: libsveltosv1beta1.ConfigurationGroupSpec{
				Action:        libsveltosv1beta1.ActionRemove,
				RequestorHash: requestorHash,
				UpdatePhase:   libsveltosv1beta1.UpdatePhaseReady,
			},
		}

		Expect(k8sClient.Create(context.TODO(), configurationGroup)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, configurationGroup)).To(Succeed())

		currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
		Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
			currentConfigurationGroup)).To(Succeed())

		// Verify APIs returns the correct status
		isBeingRemoved := pullmode.IsBeingRemoved(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, logger)
		Expect(isBeingRemoved).To(BeTrue())

		Expect(k8sClient.Update(context.TODO(), configurationGroup)).To(Succeed())
		configurationGroup.Spec.Action = libsveltosv1beta1.ActionDeploy
		Expect(k8sClient.Update(context.TODO(), configurationGroup)).To(Succeed())

		// wait for cache to sync verifying status is actually set
		Eventually(func() bool {
			currentConfigurationGroup := &libsveltosv1beta1.ConfigurationGroup{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: configurationGroup.Name},
				currentConfigurationGroup)
			if err != nil {
				return false
			}
			return currentConfigurationGroup.Spec.Action == libsveltosv1beta1.ActionDeploy
		}, time.Minute, time.Second).Should(BeTrue())

		isBeingRemoved = pullmode.IsBeingRemoved(context.TODO(), k8sClient, clusterNamespace, clusterName,
			requestorKind, requestorName, requestorFeature, logger)
		Expect(isBeingRemoved).To(BeFalse())
	})
})
