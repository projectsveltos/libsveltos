/*
Copyright 2024. projectsveltos.io. All rights reserved.

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

package sveltos_upgrade_test

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/sveltos_upgrade"
)

const (
	version = "v1.31.0"
)

// createOwnerCluster creates the CAPI Cluster or SveltosCluster object that owns the
// per-cluster version-tracking ConfigMap when the agent runs in the management cluster.
// The fake client does not auto-assign a UID, so one is set explicitly: without it, an
// assertion against OwnerReferences[0].UID would be vacuous (comparing empty to empty).
func createOwnerCluster(c client.Client, clusterNamespace, clusterName string,
	clusterType libsveltosv1beta1.ClusterType) types.UID {

	uid := types.UID(randomString())

	var obj client.Object
	if clusterType == libsveltosv1beta1.ClusterTypeSveltos {
		obj = &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      clusterName,
				UID:       uid,
			},
		}
	} else {
		obj = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterNamespace,
				Name:      clusterName,
				UID:       uid,
			},
		}
	}

	Expect(c.Create(context.TODO(), obj)).To(Succeed())
	return uid
}

var _ = Describe("SveltosAgent compatibility checks", func() {
	var logger logr.Logger
	var sveltosNamespace string

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig())
		sveltosNamespace = randomString()
	})

	It("Create ConfigMap with Sveltos-agent version", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeSveltos

		Expect(sveltos_upgrade.StoreSveltosAgentVersion(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, false, logger)).To(Succeed())

		cm := &corev1.ConfigMap{}
		Expect(c.Get(context.TODO(),
			types.NamespacedName{Namespace: sveltosNamespace, Name: sveltos_upgrade.SveltosAgentConfigMapName},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))

		ownerUID := createOwnerCluster(c, clusterNamespace, clusterName, clusterType)

		Expect(sveltos_upgrade.StoreSveltosAgentVersion(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, true, logger)).To(Succeed())

		name := sveltos_upgrade.GenerateName(sveltos_upgrade.SveltosAgentType, clusterName, clusterType)
		Expect(c.Get(context.TODO(),
			types.NamespacedName{Namespace: clusterNamespace, Name: name}, cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))
		Expect(cm.OwnerReferences).To(HaveLen(1))
		Expect(cm.OwnerReferences[0].UID).To(Equal(ownerUID))
	})

	It("Update ConfigMap with Sveltos-agent version", func() {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: sveltosNamespace,
				Name:      sveltos_upgrade.SveltosAgentConfigMapName,
			},
			Data: map[string]string{
				sveltos_upgrade.ConfigMapKey: randomString(),
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeSveltos

		Expect(sveltos_upgrade.StoreSveltosAgentVersion(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, false, logger)).To(Succeed())

		Expect(c.Get(context.TODO(),
			types.NamespacedName{
				Namespace: sveltosNamespace,
				Name:      sveltos_upgrade.SveltosAgentConfigMapName},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))

		ownerUID := createOwnerCluster(c, clusterNamespace, clusterName, clusterType)

		Expect(sveltos_upgrade.StoreSveltosAgentVersion(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, true, logger)).To(Succeed())
		name := sveltos_upgrade.GenerateName(sveltos_upgrade.SveltosAgentType, clusterName, clusterType)

		Expect(c.Get(context.TODO(),
			types.NamespacedName{
				Namespace: clusterNamespace,
				Name:      name},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))
		Expect(cm.OwnerReferences).To(HaveLen(1))
		Expect(cm.OwnerReferences[0].UID).To(Equal(ownerUID))
	})

	It("IsSveltosAgentVersionCompatible returns true Sveltos-agent version is compatible (agent in management cluster)", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeCapi

		cmInfo := sveltos_upgrade.GetSveltosAgentConfigMapInfo(sveltosNamespace, clusterNamespace, clusterName, clusterType, true)

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cmInfo.Namespace,
				Name:      cmInfo.Name,
			},
			Data: map[string]string{
				sveltos_upgrade.ConfigMapKey: version,
			},
		}
		initObjects := []client.Object{
			cm,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
		logger := textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))
		Expect(sveltos_upgrade.IsSveltosAgentVersionCompatible(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, true, logger)).To(BeTrue())
		Expect(sveltos_upgrade.IsSveltosAgentVersionCompatible(context.TODO(), c, sveltosNamespace, randomString(),
			clusterNamespace, clusterName, clusterType, true, logger)).To(BeFalse())
	})
})

var _ = Describe("DriftDetection compatibility checks", func() {
	var logger logr.Logger
	var sveltosNamespace string

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig())
		sveltosNamespace = randomString()
	})

	It("IsDriftDetectionVersionCompatible returns true when drift-detection version is compatible (agent in management cluster)", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeSveltos

		cmInfo := sveltos_upgrade.GetDriftDetectionConfigMapInfo(sveltosNamespace, clusterNamespace, clusterName, clusterType, true)

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cmInfo.Namespace,
				Name:      cmInfo.Name,
			},
			Data: map[string]string{
				sveltos_upgrade.ConfigMapKey: version,
			},
		}
		initObjects := []client.Object{
			cm,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
		logger := textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))
		Expect(sveltos_upgrade.IsDriftDetectionVersionCompatible(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, true, logger)).To(BeTrue())
		Expect(sveltos_upgrade.IsDriftDetectionVersionCompatible(context.TODO(), c, sveltosNamespace, randomString(),
			randomString(), randomString(), clusterType, true, logger)).To(BeFalse())
	})

	It("Create ConfigMap with drift-detection-manager version", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		Expect(sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, sveltosNamespace, version,
			randomString(), randomString(), libsveltosv1beta1.ClusterTypeCapi, false, logger)).To(Succeed())

		cm := &corev1.ConfigMap{}
		Expect(c.Get(context.TODO(),
			types.NamespacedName{Namespace: sveltosNamespace, Name: sveltos_upgrade.DriftDetectionConfigMapName},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeCapi

		ownerUID := createOwnerCluster(c, clusterNamespace, clusterName, clusterType)

		Expect(sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, true, logger)).To(Succeed())

		name := sveltos_upgrade.GenerateName(sveltos_upgrade.DriftDetectionType, clusterName, clusterType)
		Expect(c.Get(context.TODO(),
			types.NamespacedName{Namespace: clusterNamespace, Name: name}, cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))
		Expect(cm.OwnerReferences).To(HaveLen(1))
		Expect(cm.OwnerReferences[0].UID).To(Equal(ownerUID))
	})

	It("Update ConfigMap with drift-detection-manager version", func() {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: sveltosNamespace,
				Name:      sveltos_upgrade.DriftDetectionConfigMapName,
			},
			Data: map[string]string{
				sveltos_upgrade.ConfigMapKey: randomString(),
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		Expect(sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, sveltosNamespace, version,
			randomString(), randomString(), libsveltosv1beta1.ClusterTypeCapi, false, logger)).To(Succeed())

		Expect(c.Get(context.TODO(),
			types.NamespacedName{
				Namespace: sveltosNamespace,
				Name:      sveltos_upgrade.DriftDetectionConfigMapName},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeSveltos

		ownerUID := createOwnerCluster(c, clusterNamespace, clusterName, clusterType)

		Expect(sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, true, logger)).To(Succeed())

		name := sveltos_upgrade.GenerateName(sveltos_upgrade.DriftDetectionType, clusterName, clusterType)
		Expect(c.Get(context.TODO(),
			types.NamespacedName{
				Namespace: clusterNamespace,
				Name:      name},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))
		Expect(cm.OwnerReferences).To(HaveLen(1))
		Expect(cm.OwnerReferences[0].UID).To(Equal(ownerUID))
	})

	It("StoreDriftDetectionVersion returns NotFound and creates no ConfigMap when the owning cluster is missing", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeCapi

		err := sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, sveltosNamespace, version,
			clusterNamespace, clusterName, clusterType, true, logger)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue())

		name := sveltos_upgrade.GenerateName(sveltos_upgrade.DriftDetectionType, clusterName, clusterType)
		cm := &corev1.ConfigMap{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: clusterNamespace, Name: name}, cm)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})
})
