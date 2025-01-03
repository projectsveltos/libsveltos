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

package crd_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/projectsveltos/libsveltos/lib/crd"
)

var _ = Describe("CRD", func() {
	It("Gets the Classifier CustomResourceDefinition", func() {
		yaml := crd.GetClassifierCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_classifiers.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the ClassifierReport CustomResourceDefinition", func() {
		yaml := crd.GetClassifierReportCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_classifierreports.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the DebuggingConfiguration CustomResourceDefinition", func() {
		yaml := crd.GetDebuggingConfigurationCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_debuggingconfigurations.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the AccessRequest CustomResourceDefinition", func() {
		yaml := crd.GetAccessRequestCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_accessrequests.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the SveltosCluster CustomResourceDefinition", func() {
		yaml := crd.GetSveltosClusterCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_sveltosclusters.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the ResourceSummary CustomResourceDefinition", func() {
		yaml := crd.GetResourceSummaryCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_resourcesummaries.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the RoleRequest CustomResourceDefinition", func() {
		yaml := crd.GetRoleRequestCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_rolerequests.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the ClusterHealthCheck CustomResourceDefinition", func() {
		yaml := crd.GetClusterHealthCheckCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_clusterhealthchecks.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the HealthCheck CustomResourceDefinition", func() {
		yaml := crd.GetHealthCheckCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_healthchecks.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the HealthCheckReport CustomResourceDefinition", func() {
		yaml := crd.GetHealthCheckReportCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_healthcheckreports.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the EventSource CustomResourceDefinition", func() {
		yaml := crd.GetEventSourceCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_eventsources.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the EventReport CustomResourceDefinition", func() {
		yaml := crd.GetEventReportCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_eventreports.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the Reloader CustomResourceDefinition", func() {
		yaml := crd.GetReloaderCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_reloaders.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the ReloaderReport CustomResourceDefinition", func() {
		yaml := crd.GetReloaderReportCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_reloaderreports.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the ClusterSet CustomResourceDefinition", func() {
		yaml := crd.GetClusterSetCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_clustersets.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the Set CustomResourceDefinition", func() {
		yaml := crd.GetSetCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_sets.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Gets the Techsupport CustomResourceDefinition", func() {
		yaml := crd.GetTechsupportCRDYAML()

		filename := "../../manifests/apiextensions.k8s.io_v1_customresourcedefinition_techsupports.lib.projectsveltos.io.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})
})
