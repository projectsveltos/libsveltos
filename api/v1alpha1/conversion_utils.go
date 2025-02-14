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

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	v1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

var (
	configlog = logf.Log.WithName("conversion")
)

func convertV1Alpha1SelectorToV1Beta1(clusterSelector *Selector) (*libsveltosv1beta1.Selector, error) {
	labelSelector, err := metav1.ParseToLabelSelector(string(*clusterSelector))
	if err != nil {
		return nil, fmt.Errorf("error converting labels.Selector to metav1.Selector: %w", err)
	}
	return &libsveltosv1beta1.Selector{LabelSelector: *labelSelector}, nil
}

func convertV1Beta1SelectorToV1Alpha1(clusterSelector *libsveltosv1beta1.Selector) (Selector, error) {
	labelSelector, err := clusterSelector.ToSelector()
	if err != nil {
		return "", fmt.Errorf("failed to convert : %w", err)
	}

	return Selector(labelSelector.String()), nil
}

func Convert_v1beta1_EventReportSpec_To_v1alpha1_EventReportSpec(srcSpec *v1beta1.EventReportSpec,
	dstSpec *EventReportSpec, scope conversion.Scope) error {

	if err := autoConvert_v1beta1_EventReportSpec_To_v1alpha1_EventReportSpec(srcSpec, dstSpec, nil); err != nil {
		return err
	}

	return nil
}

func Convert_v1beta1_EventSourceSpec_To_v1alpha1_EventSourceSpec(srcSpec *v1beta1.EventSourceSpec,
	dstSpec *EventSourceSpec, scope conversion.Scope) error {

	if err := autoConvert_v1beta1_EventSourceSpec_To_v1alpha1_EventSourceSpec(srcSpec, dstSpec, nil); err != nil {
		return err
	}

	return nil
}

func Convert_v1beta1_SveltosClusterSpec_To_v1alpha1_SveltosClusterSpec(srcSpec *v1beta1.SveltosClusterSpec,
	dstSpec *SveltosClusterSpec, scope conversion.Scope) error {

	if err := autoConvert_v1beta1_SveltosClusterSpec_To_v1alpha1_SveltosClusterSpec(srcSpec, dstSpec, nil); err != nil {
		return err
	}

	return nil
}
