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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

// ConvertTo converts v1alpha1 to the Hub version (v1beta1).
func (src *ClusterSet) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*libsveltosv1beta1.ClusterSet)
	err := Convert_v1alpha1_ClusterSet_To_v1beta1_ClusterSet(src, dst, nil)
	if err != nil {
		return err
	}

	if src.Spec.ClusterSelector == "" {
		dst.Spec.ClusterSelector.LabelSelector = metav1.LabelSelector{}
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1beta1) to this v1alpha1.
func (dst *ClusterSet) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*libsveltosv1beta1.ClusterSet)
	err := Convert_v1beta1_ClusterSet_To_v1alpha1_ClusterSet(src, dst, nil)
	if err != nil {
		return err
	}

	if src.Spec.ClusterSelector.MatchLabels == nil {
		dst.Spec.ClusterSelector = ""
	}

	return nil
}
