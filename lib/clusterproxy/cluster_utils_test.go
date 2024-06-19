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

package clusterproxy_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/textlogger"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/internal/test/helpers/external"
	"github.com/projectsveltos/libsveltos/lib/clusterproxy"
	"github.com/projectsveltos/libsveltos/lib/sharding"
)

var _ = Describe("Cluster utils", func() {
	var namespace string
	var cluster *clusterv1.Cluster
	var sveltosCluster *libsveltosv1beta1.SveltosCluster

	BeforeEach(func() {
		namespace = "cluster-utils" + randomString()

		cluster = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: namespace,
			},
			Spec: clusterv1.ClusterSpec{
				Paused: true,
			},
			Status: clusterv1.ClusterStatus{
				ControlPlaneReady: true,
			},
		}

		sveltosCluster = &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: namespace,
			},
			Spec: libsveltosv1beta1.SveltosClusterSpec{
				Paused: true,
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}
	})

	It("IsClusterPaused returns true when Spec.Paused is set to true", func() {
		initObjects := []client.Object{
			cluster, sveltosCluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		paused, err := clusterproxy.IsClusterPaused(context.TODO(), c, cluster.Namespace,
			cluster.Name, libsveltosv1beta1.ClusterTypeCapi)
		Expect(err).To(BeNil())
		Expect(paused).To(BeTrue())

		paused, err = clusterproxy.IsClusterPaused(context.TODO(), c, sveltosCluster.Namespace,
			sveltosCluster.Name, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(paused).To(BeTrue())
	})

	It("IsClusterPaused returns false when Spec.Paused is set to false", func() {
		cluster.Spec.Paused = false
		sveltosCluster.Spec.Paused = false
		initObjects := []client.Object{
			cluster, sveltosCluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		paused, err := clusterproxy.IsClusterPaused(context.TODO(), c, cluster.Namespace,
			cluster.Name, libsveltosv1beta1.ClusterTypeCapi)
		Expect(err).To(BeNil())
		Expect(paused).To(BeFalse())

		paused, err = clusterproxy.IsClusterPaused(context.TODO(), c, sveltosCluster.Namespace,
			sveltosCluster.Name, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(paused).To(BeFalse())
	})

	It("GetSecretData returns kubeconfig data", func() {
		randomData := []byte(randomString())
		sveltosSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: sveltosCluster.Namespace,
				Name:      sveltosCluster.Name + "-sveltos-kubeconfig",
			},
			Data: map[string][]byte{
				"data": randomData,
			},
		}

		capiSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + "-kubeconfig",
			},
			Data: map[string][]byte{
				"data": randomData,
			},
		}

		initObjects := []client.Object{
			sveltosCluster,
			cluster,
			&sveltosSecret,
			&capiSecret,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		data, err := clusterproxy.GetSecretData(context.TODO(), c, cluster.Namespace, cluster.Name,
			"", "", libsveltosv1beta1.ClusterTypeCapi,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(data).To(Equal(randomData))

		data, err = clusterproxy.GetSecretData(context.TODO(), c, sveltosCluster.Namespace, sveltosCluster.Name,
			"", "", libsveltosv1beta1.ClusterTypeSveltos,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(data).To(Equal(randomData))
	})

	It("GetListOfClusters returns the all existing Clusters", func() {
		cluster1 := &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
			},
		}

		cluster2 := &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
			},
		}

		clusterCRD := external.TestClusterCRD.DeepCopy()

		initObjects := []client.Object{
			clusterCRD,
			cluster1,
			cluster2,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		matches, err := clusterproxy.GetListOfClusters(context.TODO(), c, "",
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(2))

		matches, err = clusterproxy.GetListOfClusters(context.TODO(), c, cluster1.Namespace,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(1))
		Expect(matches).To(ContainElement(corev1.ObjectReference{
			Namespace: cluster1.Namespace, Name: cluster1.Name,
			Kind: "Cluster", APIVersion: clusterv1.GroupVersion.String(),
		}))
	})

	It("GetListOfClustersForShardKey returns all existing Clusters with shard annotation set to provided key", func() {
		shardKey := randomString()

		cluster1 := &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				Annotations: map[string]string{
					sharding.ShardAnnotation: shardKey,
				},
			},
		}

		cluster2 := &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
				Annotations: map[string]string{
					sharding.ShardAnnotation: randomString(),
				},
			},
		}

		cluster3 := &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: randomString(),
				Name:      randomString(),
			},
		}

		clusterCRD := external.TestClusterCRD.DeepCopy()

		initObjects := []client.Object{
			clusterCRD,
			cluster1,
			cluster2,
			cluster3,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		matches, err := clusterproxy.GetListOfClustersForShardKey(context.TODO(), c, "", shardKey,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(1))
		Expect(matches).To(ContainElement(corev1.ObjectReference{
			Namespace: cluster1.Namespace, Name: cluster1.Name,
			Kind: "Cluster", APIVersion: clusterv1.GroupVersion.String(),
		}))

		matches, err = clusterproxy.GetListOfClustersForShardKey(context.TODO(), c, "", "",
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(1))
		Expect(matches).To(ContainElement(corev1.ObjectReference{
			Namespace: cluster3.Namespace, Name: cluster3.Name,
			Kind: "Cluster", APIVersion: clusterv1.GroupVersion.String(),
		}))

		matches, err = clusterproxy.GetListOfClustersForShardKey(context.TODO(), c, cluster1.Namespace, shardKey,
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(1))
		Expect(matches).To(ContainElement(corev1.ObjectReference{
			Namespace: cluster1.Namespace, Name: cluster1.Name,
			Kind: "Cluster", APIVersion: clusterv1.GroupVersion.String(),
		}))
	})

	It("GetMatchingClusters matches no cluster when Selector is empty", func() {
		selector := libsveltosv1beta1.Selector{
			LabelSelector: metav1.LabelSelector{},
		}

		sveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: randomString(),
				Labels: map[string]string{
					randomString(): randomString(),
				},
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}

		nonMatchingSveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: randomString(),
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}

		cluster.Labels = map[string]string{
			randomString(): randomString(),
		}

		initObjects := []client.Object{
			cluster,
			sveltosCluster,
			nonMatchingSveltosCluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(initObjects...).WithObjects(initObjects...).Build()

		matches, err := clusterproxy.GetMatchingClusters(context.TODO(), c, &selector.LabelSelector, "",
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(0))
	})

	It("getMatchingClusters returns matchin CAPI Cluster", func() {
		selector := libsveltosv1beta1.Selector{
			LabelSelector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "env",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"qa"},
					},
					{
						Key:      "zone",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"west"},
					},
				},
			},
		}

		currentLabels := map[string]string{
			"env":  "qa",
			"zone": "west",
		}

		sveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: randomString(),
				Labels:    currentLabels,
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}

		nonMatchingSveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: randomString(),
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}

		cluster.Labels = currentLabels

		initObjects := []client.Object{
			cluster,
			sveltosCluster,
			nonMatchingSveltosCluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(initObjects...).WithObjects(initObjects...).Build()

		matches, err := clusterproxy.GetMatchingClusters(context.TODO(), c, &selector.LabelSelector, "",
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(2))
		Expect(matches).To(ContainElement(
			corev1.ObjectReference{Namespace: cluster.Namespace, Name: cluster.Name,
				Kind: "Cluster", APIVersion: clusterv1.GroupVersion.String()}))
		Expect(matches).To(ContainElement(
			corev1.ObjectReference{Namespace: sveltosCluster.Namespace, Name: sveltosCluster.Name,
				Kind: libsveltosv1beta1.SveltosClusterKind, APIVersion: libsveltosv1beta1.GroupVersion.String()}))

		matches, err = clusterproxy.GetMatchingClusters(context.TODO(), c, &selector.LabelSelector,
			sveltosCluster.Namespace, textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(1))
		Expect(matches).To(ContainElement(
			corev1.ObjectReference{Namespace: sveltosCluster.Namespace, Name: sveltosCluster.Name,
				Kind: libsveltosv1beta1.SveltosClusterKind, APIVersion: libsveltosv1beta1.GroupVersion.String()}))

	})

	It("getMatchingClusters returns matchin CAPI Cluster", func() {
		key1 := randomString()
		value1 := randomString()
		key2 := randomString()
		value2 := randomString()

		selector := libsveltosv1beta1.Selector{
			LabelSelector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      key1,
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{value1},
					},
					{
						Key:      key2,
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{value2},
					},
				},
			},
		}

		currentLabels := map[string]string{
			key1: value1,
			key2: value2,
		}

		sveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: randomString(),
				Labels:    currentLabels,
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}

		nonMatchingSveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: randomString(),
				Labels: map[string]string{
					randomString(): randomString(),
					key1:           value1,
				},
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}

		cluster.Labels = currentLabels

		initObjects := []client.Object{
			cluster,
			sveltosCluster,
			nonMatchingSveltosCluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(initObjects...).WithObjects(initObjects...).Build()

		matches, err := clusterproxy.GetMatchingClusters(context.TODO(), c, &selector.LabelSelector, "",
			textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(2))
		Expect(matches).To(ContainElement(
			corev1.ObjectReference{Namespace: cluster.Namespace, Name: cluster.Name,
				Kind: "Cluster", APIVersion: clusterv1.GroupVersion.String()}))
		Expect(matches).To(ContainElement(
			corev1.ObjectReference{Namespace: sveltosCluster.Namespace, Name: sveltosCluster.Name,
				Kind: libsveltosv1beta1.SveltosClusterKind, APIVersion: libsveltosv1beta1.GroupVersion.String()}))

		matches, err = clusterproxy.GetMatchingClusters(context.TODO(), c, &selector.LabelSelector,
			sveltosCluster.Namespace, textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1))))
		Expect(err).To(BeNil())
		Expect(len(matches)).To(Equal(1))
		Expect(matches).To(ContainElement(
			corev1.ObjectReference{Namespace: sveltosCluster.Namespace, Name: sveltosCluster.Name,
				Kind: libsveltosv1beta1.SveltosClusterKind, APIVersion: libsveltosv1beta1.GroupVersion.String()}))

	})
})
