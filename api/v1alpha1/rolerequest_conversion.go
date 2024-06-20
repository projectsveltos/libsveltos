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
	apimachineryconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

// ConvertTo converts v1alpha1 to the Hub version (v1beta1).
func (src *RoleRequest) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*libsveltosv1beta1.RoleRequest)
	err := Convert_v1alpha1_RoleRequest_To_v1beta1_RoleRequest(src, dst, nil)
	if err != nil {
		return err
	}

	if src.Spec.ClusterSelector == "" {
		dst.Spec.ClusterSelector.LabelSelector = metav1.LabelSelector{}
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1beta1) to this v1alpha1.
func (dst *RoleRequest) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*libsveltosv1beta1.RoleRequest)
	err := Convert_v1beta1_RoleRequest_To_v1alpha1_RoleRequest(src, dst, nil)
	if err != nil {
		return err
	}

	if src.Spec.ClusterSelector.MatchLabels == nil {
		dst.Spec.ClusterSelector = ""
	}

	return nil
}

func Convert_v1alpha1_RoleRequestSpec_To_v1beta1_RoleRequestSpec(srcSpec *RoleRequestSpec,
	dstSpec *libsveltosv1beta1.RoleRequestSpec, scope apimachineryconversion.Scope,
) error {

	if err := autoConvert_v1alpha1_RoleRequestSpec_To_v1beta1_RoleRequestSpec(srcSpec, dstSpec, nil); err != nil {
		return err
	}

	selector, err := convertV1Alpha1SelectorToV1Beta1(&srcSpec.ClusterSelector)
	if err != nil {
		configlog.V(logs.LogInfo).Info(fmt.Sprintf("failed to convert ClusterSelector: %v", err))
		return err
	}

	dstSpec.ClusterSelector = *selector

	return nil
}

func Convert_v1beta1_RoleRequestSpec_To_v1alpha1_RoleRequestSpec(srcSpec *libsveltosv1beta1.RoleRequestSpec,
	dstSpec *RoleRequestSpec, scope apimachineryconversion.Scope,
) error {

	if err := autoConvert_v1beta1_RoleRequestSpec_To_v1alpha1_RoleRequestSpec(srcSpec, dstSpec, nil); err != nil {
		return err
	}

	selector, err := convertV1Beta1SelectorToV1Alpha1(&srcSpec.ClusterSelector)
	if err != nil {
		configlog.V(logs.LogInfo).Info(fmt.Sprintf("failed to convert ClusterSelector: %v", err))
		return err
	}

	dstSpec.ClusterSelector = selector

	return nil
}
