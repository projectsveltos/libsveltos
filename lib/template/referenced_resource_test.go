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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/template"
)

var _ = Describe("Template", func() {
	It("getReferenceResourceNamespace returns the referenced resource namespace when set. cluster namespace otherwise.", func() {
		clusterNamespace := randomString()

		// When namespace is not set, cluster namespace is returned
		namespace := ""
		Expect(template.GetReferenceResourceNamespace(clusterNamespace, namespace)).To(Equal(clusterNamespace))

		// When namespace is set, namespace is returned
		namespace = randomString()
		Expect(template.GetReferenceResourceNamespace(clusterNamespace, namespace)).To(Equal(namespace))
	})

	It("getReferenceResourceName instantiate template using cluster data.", func() {
		name := randomString()

		clusterNamespace := randomString()
		clusterName := randomString()
		clusterKind := string(libsveltosv1beta1.SveltosClusterKind)

		// If name is not expressed as a template, name is returned
		Expect(template.GetReferenceResourceName(clusterNamespace, clusterName, clusterKind, name)).To(Equal(name))

		name = "test-{{ .Cluster.metadata.namespace }}--{{ .Cluster.metadata.name}}"
		Expect(template.GetReferenceResourceName(clusterNamespace, clusterName, clusterKind, name)).To(
			Equal(fmt.Sprintf("test-%s--%s", clusterNamespace, clusterName)))
	})

})
