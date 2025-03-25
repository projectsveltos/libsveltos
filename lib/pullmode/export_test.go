/*
Copyright 2023. projectsveltos.io. All rights reserved.

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
package pullmode

var (
	GetConfigurationBundleLabels    = getConfigurationBundleLabels
	CreateConfigurationBundle       = createConfigurationBundle
	UpdateConfigurationBundle       = updateConfigurationBundle
	ReconcileConfigurationBundle    = reconcileConfigurationBundle
	GetStagedConfigurationBundles   = getStagedConfigurationBundles
	DeleteStaleConfigurationBundles = deleteStaleConfigurationBundles

	GetConfigurationGroupLabels = getConfigurationGroupLabels
	CreateConfigurationGroup    = createConfigurationGroup
	UpdateConfigurationGroup    = updateConfigurationGroup
	ReconcileConfigurationGroup = reconcileConfigurationGroup
)

const (
	StagedLabelKey   = stagedLabelKey
	StagedLabelValue = stagedLabelValue
)

type (
	BundleData = bundleData
)
