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

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

// ConvertTo converts v1alpha1 to the Hub version (v1beta1).
func (src *ClusterSet) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*libsveltosv1beta1.ClusterSet)

	configlog.V(logs.LogInfo).Info("convert ClusterSet from v1alpha1 to v1beta1")

	dst.ObjectMeta = src.ObjectMeta

	err := convertV1Alpha1SetSpecToV1Beta1(&src.Spec, &dst.Spec)
	if err != nil {
		return fmt.Errorf("error converting Spec: %w", err)
	}

	err = convertV1Alpha1SetStatusToV1Beta1(&src.Status, &dst.Status)
	if err != nil {
		return fmt.Errorf("error converting Spec: %w", err)
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1beta1) to this v1alpha1.
func (dst *ClusterSet) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*libsveltosv1beta1.ClusterSet)

	configlog.V(logs.LogInfo).Info("convert ClusterSet from v1beta1 to v1alpha1")

	dst.ObjectMeta = src.ObjectMeta

	err := convertV1Beta1SetSpecToV1Alpha1(&src.Spec, &dst.Spec)
	if err != nil {
		return fmt.Errorf("error converting Spec: %w", err)
	}

	err = convertV1Beta1SetStatusToV1Alpha1(&src.Status, &dst.Status)
	if err != nil {
		return fmt.Errorf("error converting Status: %w", err)
	}

	return nil
}
