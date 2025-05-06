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

package template_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/template"
)

var _ = Describe("Template", func() {
	var scheme *runtime.Scheme
	var sveltosCluster *libsveltosv1beta1.SveltosCluster

	BeforeEach(func() {
		sveltosCluster = &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: randomString(),
			},
			Spec: libsveltosv1beta1.SveltosClusterSpec{
				Paused: false,
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}

		scheme = runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(libsveltosv1beta1.AddToScheme(scheme)).To(Succeed())
		Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	})

	It("getReferenceResourceNamespace returns the referenced resource namespace when set. cluster namespace otherwise.", func() {
		zoneKey := "zone"
		zoneValue := "us-east1"

		sveltosCluster.Annotations = map[string]string{
			zoneKey: zoneValue,
		}

		initObjects := []client.Object{
			sveltosCluster,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		// When namespace is not set, cluster namespace is returned
		referencedNamespace := ""
		instantiatedNamespace, err := template.GetReferenceResourceNamespace(context.TODO(), c,
			sveltosCluster.Namespace, sveltosCluster.Name, referencedNamespace, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(instantiatedNamespace).To(Equal(sveltosCluster.Namespace))

		// When namespace is set and not a template, namespace is returned
		referencedNamespace = randomString()
		instantiatedNamespace, err = template.GetReferenceResourceNamespace(context.TODO(), c,
			sveltosCluster.Namespace, sveltosCluster.Name, referencedNamespace, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(instantiatedNamespace).To(Equal(referencedNamespace))

		// When namespace is set and is a template, instantiated value is returned
		referencedNamespace = fmt.Sprintf("{{.Cluster.metadata.name}}-{{ index .Cluster.metadata.annotations %q }}", zoneKey)
		instantiatedNamespace, err = template.GetReferenceResourceNamespace(context.TODO(), c,
			sveltosCluster.Namespace, sveltosCluster.Name, referencedNamespace, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		expectedNamespace := fmt.Sprintf("%s-%s", sveltosCluster.Name, zoneValue)
		Expect(instantiatedNamespace).To(Equal(expectedNamespace))
	})

	It("getReferenceResourceName instantiate template using cluster data.", func() {
		envKey := "env"
		envValue := "staging"

		sveltosCluster := &libsveltosv1beta1.SveltosCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: randomString(),
				Labels: map[string]string{
					envKey: envValue,
				},
			},
			Spec: libsveltosv1beta1.SveltosClusterSpec{
				Paused: true,
			},
			Status: libsveltosv1beta1.SveltosClusterStatus{
				Ready: true,
			},
		}

		initObjects := []client.Object{
			sveltosCluster,
		}

		scheme := runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(libsveltosv1beta1.AddToScheme(scheme)).To(Succeed())
		Expect(clusterv1.AddToScheme(scheme)).To(Succeed())

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		// If name is not expressed as a template, name is returned
		name := randomString()
		instantiatedName, err := template.GetReferenceResourceName(context.TODO(), c, sveltosCluster.Namespace,
			sveltosCluster.Name, name, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		Expect(instantiatedName).To(Equal(name))

		// If name is expressed as a template, instantiated value is returned
		name = fmt.Sprintf("env-{{ index .Cluster.metadata.labels %q }}", envKey)
		instantiatedName, err = template.GetReferenceResourceName(context.TODO(), c, sveltosCluster.Namespace,
			sveltosCluster.Name, name, libsveltosv1beta1.ClusterTypeSveltos)
		Expect(err).To(BeNil())
		expectedName := fmt.Sprintf("env-%s", envValue)
		Expect(instantiatedName).To(Equal(expectedName))
	})

})
