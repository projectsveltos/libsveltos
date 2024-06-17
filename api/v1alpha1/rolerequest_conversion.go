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
	"encoding/json"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

// ConvertTo converts v1alpha1 to the Hub version (v1beta1).
func (src *RoleRequest) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*libsveltosv1beta1.RoleRequest)

	configlog.V(logs.LogInfo).Info("convert RoleRequest from v1alpha1 to v1beta1")

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.ExpirationSeconds = src.Spec.ExpirationSeconds

	jsonData, err := json.Marshal(src.Spec.RoleRefs) // Marshal the RoleRefs field
	if err != nil {
		return fmt.Errorf("error marshaling Spec.RoleRefs: %w", err)
	}
	err = json.Unmarshal(jsonData, &dst.Spec.RoleRefs) // Unmarshal to v1beta1 type
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	dst.Spec.ServiceAccountName = src.Spec.ServiceAccountName
	dst.Spec.ServiceAccountNamespace = src.Spec.ServiceAccountNamespace

	selector, err := convertV1Alpha1SelectorToV1Beta1(&src.Spec.ClusterSelector)
	if err != nil {
		configlog.V(logs.LogInfo).Info(fmt.Sprintf("failed to convert ClusterSelector: %v", err))
		return err
	}

	dst.Spec.ClusterSelector = *selector

	jsonData, err = json.Marshal(src.Status) // Marshal the Status field
	if err != nil {
		return fmt.Errorf("error marshaling Status: %w", err)
	}

	err = json.Unmarshal(jsonData, &dst.Status) // Unmarshal to v1beta1 type
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1beta1) to this v1alpha1.
func (dst *RoleRequest) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*libsveltosv1beta1.RoleRequest)

	configlog.V(logs.LogInfo).Info("convert RoleRequest from v1beta1 to v1alpha1")

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.ExpirationSeconds = src.Spec.ExpirationSeconds

	jsonData, err := json.Marshal(src.Spec.RoleRefs) // Marshal the RoleRefs field
	if err != nil {
		return fmt.Errorf("error marshaling Spec.RoleRefs: %w", err)
	}
	err = json.Unmarshal(jsonData, &dst.Spec.RoleRefs) // Unmarshal to v1beta1 type
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	dst.Spec.ServiceAccountName = src.Spec.ServiceAccountName
	dst.Spec.ServiceAccountNamespace = src.Spec.ServiceAccountNamespace

	selector, err := convertV1Beta1SelectorToV1Alpha1(&src.Spec.ClusterSelector)
	if err != nil {
		configlog.V(logs.LogInfo).Info(fmt.Sprintf("failed to convert ClusterSelector: %v", err))
		return err
	}

	dst.Spec.ClusterSelector = selector

	jsonData, err = json.Marshal(src.Status) // Marshal the Status field
	if err != nil {
		return fmt.Errorf("error marshaling Status: %w", err)
	}

	err = json.Unmarshal(jsonData, &dst.Status) // Unmarshal to v1beta1 type
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return nil
}
