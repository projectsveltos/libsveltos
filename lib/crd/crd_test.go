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
	It("Should get the Classifier CustomResourceDefinition", func() {
		yaml := crd.GetClassifierCRDYAML()

		filename := "../../config/crd/bases/lib.projectsveltos.io_classifiers.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Should get the ClassifierReport CustomResourceDefinition", func() {
		yaml := crd.GetClassifierReportCRDYAML()

		filename := "../../config/crd/bases/lib.projectsveltos.io_classifierreports.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})

	It("Should get the DebuggingConfiguration CustomResourceDefinition", func() {
		yaml := crd.GetDebuggingConfigurationCRDYAML()

		filename := "../../config/crd/bases/lib.projectsveltos.io_debuggingconfigurations.yaml"
		currentFile, err := os.ReadFile(filename)
		Expect(err).To(BeNil())

		Expect(string(yaml)).To(Equal(string(currentFile)))
	})
})
