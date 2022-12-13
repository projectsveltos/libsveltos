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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/clusterproxy"
)

const (
	upstreamClusterNamePrefix = "upstream-cluster"
)

func setupScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	if err := clusterv1.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := clientgoscheme.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := libsveltosv1alpha1.AddToScheme(s); err != nil {
		return nil, err
	}
	return s, nil
}

var _ = Describe("clusterproxy ", func() {
	var logger logr.Logger
	var cluster *clusterv1.Cluster
	var sveltosCluster *libsveltosv1alpha1.SveltosCluster
	var namespace string
	var scheme *runtime.Scheme

	BeforeEach(func() {
		var err error
		scheme, err = setupScheme()
		Expect(err).ToNot(HaveOccurred())

		logger = klogr.New()

		namespace = "reconcile" + randomString()

		logger = klogr.New()
		cluster = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      upstreamClusterNamePrefix + randomString(),
				Namespace: namespace,
				Labels: map[string]string{
					"dc": "eng",
				},
			},
		}

		sveltosCluster = &libsveltosv1alpha1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      upstreamClusterNamePrefix + randomString(),
				Namespace: namespace,
				Labels: map[string]string{
					"dc": "eng",
				},
			},
		}
	})

	It("getCAPISecretData returns an error when cluster does not exist", func() {
		initObjects := []client.Object{}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		_, err := clusterproxy.GetCAPISecretData(context.TODO(), logger, c, cluster.Namespace, cluster.Name)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Cluster %s/%s does not exist", cluster.Namespace, cluster.Name)))
	})

	It("getCAPISecretData returns an error when secret does not exist", func() {
		initObjects := []client.Object{
			cluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		_, err := clusterproxy.GetCAPISecretData(context.TODO(), logger, c, cluster.Namespace, cluster.Name)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Failed to get secret %s/%s%s", cluster.Namespace, cluster.Name,
			clusterproxy.CapiKubeconfigSecretNamePostfix)))
	})

	It("getCAPISecretData returns secret data", func() {
		randomData := []byte(randomString())
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + clusterproxy.CapiKubeconfigSecretNamePostfix,
			},
			Data: map[string][]byte{
				"data": randomData,
			},
		}

		initObjects := []client.Object{
			cluster,
			&secret,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		data, err := clusterproxy.GetCAPISecretData(context.TODO(), logger, c, cluster.Namespace, cluster.Name)
		Expect(err).To(BeNil())
		Expect(data).To(Equal(randomData))
	})

	It("getCAPIKubernetesClient returns client to access CAPI cluster", func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(testEnv.Create(context.TODO(), ns)).To(Succeed())
		Expect(testEnv.Create(context.TODO(), cluster)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + clusterproxy.CapiKubeconfigSecretNamePostfix,
			},
			Data: map[string][]byte{
				"data": testEnv.Kubeconfig,
			},
		}

		Expect(testEnv.Create(context.TODO(), secret)).To(Succeed())

		const timeout = 20 * time.Second
		// Eventual loop so testEnv Cache is synced
		Eventually(func() error {
			currentSecret := &corev1.Secret{}
			return testEnv.Get(context.TODO(),
				types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}, currentSecret)
		}, timeout, time.Second).Should(BeNil())

		wcClient, err := clusterproxy.GetCAPIKubernetesClient(context.TODO(), logger, testEnv.Client, runtime.NewScheme(),
			cluster.Namespace, cluster.Name)
		Expect(err).To(BeNil())
		Expect(wcClient).ToNot(BeNil())
	})

	It("getMachinesForCluster returns list of all machines for a CPI cluster", func() {
		cpMachine := &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + randomString(),
				Labels: map[string]string{
					clusterv1.ClusterLabelName:             cluster.Name,
					clusterv1.MachineControlPlaneLabelName: "ok",
				},
			},
		}
		workerMachine := &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + randomString(),
				Labels: map[string]string{
					clusterv1.ClusterLabelName: cluster.Name,
				},
			},
		}

		initObjects := []client.Object{
			workerMachine,
			cpMachine,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		cps, err := clusterproxy.GetMachinesForCluster(context.TODO(), c,
			&corev1.ObjectReference{Namespace: cluster.Namespace, Name: cluster.Name}, klogr.New())
		Expect(err).To(BeNil())
		Expect(len(cps.Items)).To(Equal(2))
	})

	It("IsCAPIClusterReadyToBeConfigured returns true for a cluster with one control plane machine in running phase", func() {
		cpMachine := &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + randomString(),
				Labels: map[string]string{
					clusterv1.ClusterLabelName:             cluster.Name,
					clusterv1.MachineControlPlaneLabelName: "ok",
				},
			},
		}
		cpMachine.Status.SetTypedPhase(clusterv1.MachinePhaseRunning)

		workerMachine := &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + randomString(),
				Labels: map[string]string{
					clusterv1.ClusterLabelName: cluster.Name,
				},
			},
		}
		initObjects := []client.Object{
			workerMachine,
			cpMachine,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		ready, err := clusterproxy.IsClusterReadyToBeConfigured(context.TODO(), c,
			&corev1.ObjectReference{Namespace: cluster.Namespace, Name: cluster.Name, Kind: "Cluster"}, klogr.New())
		Expect(err).To(BeNil())
		Expect(ready).To(Equal(true))
	})

	It("IsCAPIClusterReadyToBeConfigured returns false for a cluster with no control plane machine in running phase", func() {
		cpMachine := &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + randomString(),
				Labels: map[string]string{
					clusterv1.ClusterLabelName:             cluster.Name,
					clusterv1.MachineControlPlaneLabelName: "ok",
				},
			},
		}
		workerMachine := &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + randomString(),
				Labels: map[string]string{
					clusterv1.ClusterLabelName: cluster.Name,
				},
			},
			Status: clusterv1.MachineStatus{
				Phase: "Runnning",
			},
		}
		initObjects := []client.Object{
			workerMachine,
			cpMachine,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		ready, err := clusterproxy.IsClusterReadyToBeConfigured(context.TODO(), c,
			&corev1.ObjectReference{Namespace: cluster.Namespace, Name: cluster.Name}, klogr.New())
		Expect(err).To(BeNil())
		Expect(ready).To(Equal(false))
	})

	It("getSveltosSecretData returns an error when cluster does not exist", func() {
		initObjects := []client.Object{}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		_, err := clusterproxy.GetSveltosSecretData(context.TODO(), logger, c, sveltosCluster.Namespace, sveltosCluster.Name)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("SveltosCluster %s/%s does not exist",
			sveltosCluster.Namespace, sveltosCluster.Name)))
	})

	It("getSveltosSecretData returns an error when secret does not exist", func() {
		initObjects := []client.Object{
			sveltosCluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		_, err := clusterproxy.GetSveltosSecretData(context.TODO(), logger, c, sveltosCluster.Namespace, sveltosCluster.Name)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Failed to get secret %s/%s%s", sveltosCluster.Namespace, sveltosCluster.Name,
			clusterproxy.SveltosKubeconfigSecretNamePostfix)))
	})

	It("getSveltosSecretData returns secret data", func() {
		randomData := []byte(randomString())
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: sveltosCluster.Namespace,
				Name:      sveltosCluster.Name + clusterproxy.SveltosKubeconfigSecretNamePostfix,
			},
			Data: map[string][]byte{
				"data": randomData,
			},
		}

		initObjects := []client.Object{
			sveltosCluster,
			&secret,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		data, err := clusterproxy.GetSveltosSecretData(context.TODO(), logger, c, sveltosCluster.Namespace, sveltosCluster.Name)
		Expect(err).To(BeNil())
		Expect(data).To(Equal(randomData))
	})

	It("IsClusterReadyToBeConfigured returns false when Status.Ready is set to false", func() {
		sveltosCluster.Status.Ready = true
		initObjects := []client.Object{
			sveltosCluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		ready, err := clusterproxy.IsClusterReadyToBeConfigured(context.TODO(), c,
			&corev1.ObjectReference{Namespace: sveltosCluster.Namespace, Name: sveltosCluster.Name, Kind: libsveltosv1alpha1.SveltosClusterKind},
			klogr.New())
		Expect(err).To(BeNil())
		Expect(ready).To(Equal(true))

		sveltosCluster.Status.Ready = false
		Expect(c.Status().Update(context.TODO(), sveltosCluster)).To(Succeed())

		ready, err = clusterproxy.IsClusterReadyToBeConfigured(context.TODO(), c,
			&corev1.ObjectReference{Namespace: sveltosCluster.Namespace, Name: sveltosCluster.Name, Kind: libsveltosv1alpha1.SveltosClusterKind},
			klogr.New())
		Expect(err).To(BeNil())
		Expect(ready).To(Equal(false))
	})
})
