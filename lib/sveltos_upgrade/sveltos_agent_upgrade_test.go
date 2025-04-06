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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/sveltos_upgrade"
)

const (
	version = "v1.31.0"
)

var _ = Describe("SveltosAgent compatibility checks", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig())
	})

	It("Create ConfigMap with Sveltos-agent version", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeSveltos

		Expect(sveltos_upgrade.StoreSveltosAgentVersion(context.TODO(), c, version, clusterNamespace,
			clusterName, clusterType, false, logger)).To(Succeed())

		cm := &corev1.ConfigMap{}
		Expect(c.Get(context.TODO(),
			types.NamespacedName{Namespace: sveltos_upgrade.ConfigMapNamespace, Name: sveltos_upgrade.SveltosAgentConfigMapName},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))

		Expect(sveltos_upgrade.StoreSveltosAgentVersion(context.TODO(), c, version, clusterNamespace,
			clusterName, clusterType, true, logger)).To(Succeed())

		name := sveltos_upgrade.GenerateName(sveltos_upgrade.SveltosAgentType, clusterName, clusterType)
		Expect(c.Get(context.TODO(),
			types.NamespacedName{Namespace: clusterNamespace, Name: name}, cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))
	})

	It("Update ConfigMap with Sveltos-agent version", func() {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: sveltos_upgrade.ConfigMapNamespace,
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

		Expect(sveltos_upgrade.StoreSveltosAgentVersion(context.TODO(), c, version, clusterNamespace, clusterName,
			clusterType, false, logger)).To(Succeed())

		Expect(c.Get(context.TODO(),
			types.NamespacedName{
				Namespace: sveltos_upgrade.ConfigMapNamespace,
				Name:      sveltos_upgrade.SveltosAgentConfigMapName},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))

		Expect(sveltos_upgrade.StoreSveltosAgentVersion(context.TODO(), c, version, clusterNamespace, clusterName,
			clusterType, true, logger)).To(Succeed())
		name := sveltos_upgrade.GenerateName(sveltos_upgrade.SveltosAgentType, clusterName, clusterType)

		Expect(c.Get(context.TODO(),
			types.NamespacedName{
				Namespace: clusterNamespace,
				Name:      name},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))
	})

	It("IsSveltosAgentVersionCompatible returns true Sveltos-agent version is compatible (agent in management cluster)", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeCapi

		cmInfo := sveltos_upgrade.GetSveltosAgentConfigMapInfo(clusterNamespace, clusterName, clusterType, true)

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
		Expect(sveltos_upgrade.IsSveltosAgentVersionCompatible(context.TODO(), c, version, clusterNamespace, clusterName,
			clusterType, true, logger)).To(BeTrue())
		Expect(sveltos_upgrade.IsSveltosAgentVersionCompatible(context.TODO(), c, randomString(), clusterNamespace, clusterName,
			clusterType, true, logger)).To(BeFalse())
	})
})

var _ = Describe("DriftDetection compatibility checks", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig())
	})

	It("IsDriftDetectionVersionCompatible returns true when drift-detection version is compatible (agent in management cluster)", func() {
		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeSveltos

		cmInfo := sveltos_upgrade.GetDriftDetectionConfigMapInfo(clusterNamespace, clusterName, clusterType, true)

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
		Expect(sveltos_upgrade.IsDriftDetectionVersionCompatible(context.TODO(), c, version, clusterNamespace, clusterName,
			clusterType, true, logger)).To(BeTrue())
		Expect(sveltos_upgrade.IsDriftDetectionVersionCompatible(context.TODO(), c, randomString(), randomString(), randomString(),
			clusterType, true, logger)).To(BeFalse())
	})

	It("Create ConfigMap with drift-detection-manager version", func() {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		Expect(sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, version, randomString(), randomString(),
			libsveltosv1beta1.ClusterTypeCapi, false, logger)).To(Succeed())

		cm := &corev1.ConfigMap{}
		Expect(c.Get(context.TODO(),
			types.NamespacedName{Namespace: sveltos_upgrade.ConfigMapNamespace, Name: sveltos_upgrade.DriftDetectionConfigMapName},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeCapi

		Expect(sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, version, clusterNamespace, clusterName,
			clusterType, true, logger)).To(Succeed())

		name := sveltos_upgrade.GenerateName(sveltos_upgrade.DriftDetectionType, clusterName, clusterType)
		Expect(c.Get(context.TODO(),
			types.NamespacedName{Namespace: clusterNamespace, Name: name}, cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))
	})

	It("Update ConfigMap with drift-detection-manager version", func() {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: sveltos_upgrade.ConfigMapNamespace,
				Name:      sveltos_upgrade.DriftDetectionConfigMapName,
			},
			Data: map[string]string{
				sveltos_upgrade.ConfigMapKey: randomString(),
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		Expect(sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, version, randomString(), randomString(),
			libsveltosv1beta1.ClusterTypeCapi, false, logger)).To(Succeed())

		Expect(c.Get(context.TODO(),
			types.NamespacedName{
				Namespace: sveltos_upgrade.ConfigMapNamespace,
				Name:      sveltos_upgrade.DriftDetectionConfigMapName},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterType := libsveltosv1beta1.ClusterTypeSveltos

		Expect(sveltos_upgrade.StoreDriftDetectionVersion(context.TODO(), c, version, clusterNamespace, clusterName,
			clusterType, true, logger)).To(Succeed())

		name := sveltos_upgrade.GenerateName(sveltos_upgrade.DriftDetectionType, clusterName, clusterType)
		Expect(c.Get(context.TODO(),
			types.NamespacedName{
				Namespace: clusterNamespace,
				Name:      name},
			cm)).To(Succeed())
		Expect(cm.Data).ToNot(BeNil())
		Expect(cm.Data[sveltos_upgrade.ConfigMapKey]).To(Equal(version))
	})
})
