//go:build ignore
// +build ignore

/*
Copyright 2022-23. projectsveltos.io. All rights reserved.

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

package main

import (
	"os"
	"path/filepath"
	"text/template"
)

const (
	crdTemplate = `// Generated by *go generate* - DO NOT EDIT
/*
Copyright 2022-23. projectsveltos.io. All rights reserved.

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
package crd

var {{ .ExportedVar }}File = {{ printf "%q" .File }}
var {{ .ExportedVar }}CRD = []byte({{- printf "%s" .CRD -}})
`
)

func generate(filename, outputFilename, crdName string) {
	// Get CustomResourceDefinition file
	fileAbs, err := filepath.Abs(filename)
	if err != nil {
		panic(err)
	}

	content, err := os.ReadFile(fileAbs)
	if err != nil {
		panic(err)
	}
	contentStr := "`" + string(content) + "`"

	// Find the output.
	crd, err := os.Create(outputFilename + ".go")
	if err != nil {
		panic(err)
	}
	defer crd.Close()

	// Store file contents.
	type CRDInfo struct {
		CRD         string
		File        string
		ExportedVar string
	}
	mi := CRDInfo{
		CRD:         contentStr,
		File:        filename,
		ExportedVar: crdName,
	}

	// Generate template.
	manifest := template.Must(template.New("crd-generate").Parse(crdTemplate))
	if err := manifest.Execute(crd, mi); err != nil {
		panic(err)
	}
}

func main() {
	classifierFile := "../../config/crd/bases/lib.projectsveltos.io_classifiers.yaml"
	generate(classifierFile, "classifiers", "Classifier")

	classifierReportFile := "../../config/crd/bases/lib.projectsveltos.io_classifierreports.yaml"
	generate(classifierReportFile, "classifierreports", "ClassifierReport")

	debuggingConfigurationFile := "../../config/crd/bases/lib.projectsveltos.io_debuggingconfigurations.yaml"
	generate(debuggingConfigurationFile, "debuggingconfigurations", "DebuggingConfiguration")

	accessRequestFile := "../../config/crd/bases/lib.projectsveltos.io_accessrequests.yaml"
	generate(accessRequestFile, "accessrequests", "AccessRequest")

	sveltosClusterFile := "../../config/crd/bases/lib.projectsveltos.io_sveltosclusters.yaml"
	generate(sveltosClusterFile, "sveltosclusters", "SveltosCluster")

	resourceSummaryFile := "../../config/crd/bases/lib.projectsveltos.io_resourcesummaries.yaml"
	generate(resourceSummaryFile, "resourcesummaries", "ResourceSummary")

	roleRequestFile := "../../config/crd/bases/lib.projectsveltos.io_rolerequests.yaml"
	generate(roleRequestFile, "rolerequests", "RoleRequest")

	clusterHealthCheckFile := "../../config/crd/bases/lib.projectsveltos.io_clusterhealthchecks.yaml"
	generate(clusterHealthCheckFile, "clusterhealthchecks", "ClusterHealthCheck")

	healthCheckFile := "../../config/crd/bases/lib.projectsveltos.io_healthchecks.yaml"
	generate(healthCheckFile, "healthchecks", "HealthCheck")

	healthCheckReportFile := "../../config/crd/bases/lib.projectsveltos.io_healthcheckreports.yaml"
	generate(healthCheckReportFile, "healthcheckreports", "HealthCheckReport")

	eventSourceFile := "../../config/crd/bases/lib.projectsveltos.io_eventsources.yaml"
	generate(eventSourceFile, "eventsources", "EventSource")

	clusterEventSourceFile := "../../config/crd/bases/lib.projectsveltos.io_clustereventsources.yaml"
	generate(clusterEventSourceFile, "clustereventsources", "ClusterEventSource")

	eventReportFile := "../../config/crd/bases/lib.projectsveltos.io_eventreports.yaml"
	generate(eventReportFile, "eventreports", "EventReport")

	addonComplianceFile := "../../config/crd/bases/lib.projectsveltos.io_addoncompliances.yaml"
	generate(addonComplianceFile, "addoncompliances", "AddonCompliance")

	reloaderFile := "../../config/crd/bases/lib.projectsveltos.io_reloaders.yaml"
	generate(reloaderFile, "reloaders", "Reloader")

	reloaderReportFile := "../../config/crd/bases/lib.projectsveltos.io_reloaderreports.yaml"
	generate(reloaderReportFile, "reloaderreports", "ReloaderReport")

}
