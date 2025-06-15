/*
Copyright 2025. projectsveltos.io. All rights reserved.

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

package deployer

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

func UndeployStaleResource(ctx context.Context, skipAnnotationKey, skipAnnotationValue string, c client.Client,
	profile client.Object, leavePolicies, isDryRunMode bool, r unstructured.Unstructured,
	currentPolicies map[string]libsveltosv1beta1.Resource, logger logr.Logger) (*libsveltosv1beta1.ResourceReport, error) {

	logger.V(logs.LogVerbose).Info(fmt.Sprintf("considering %s/%s", r.GetNamespace(), r.GetName()))

	// Verify if this policy was deployed because of a projectsveltos (ReferenceLabelName
	// is present as label in such a case).
	if !hasLabel(&r, ReferenceNameLabel, "") {
		return nil, nil
	}

	if skipAnnotationKey != "" {
		if !hasAnnotation(&r, skipAnnotationKey, skipAnnotationValue) {
			return nil, nil
		}
	}

	var resourceReport *libsveltosv1beta1.ResourceReport = nil
	// If in DryRun do not withdrawn any policy.
	// If this ClusterSummary is the only OwnerReference and it is not deploying this policy anymore,
	// policy would be withdrawn
	if isDryRunMode {
		if canDelete(&r, currentPolicies) && isResourceOwner(&r, profile) &&
			!leavePolicies {

			resourceReport = &libsveltosv1beta1.ResourceReport{
				Resource: libsveltosv1beta1.Resource{
					Kind: r.GetObjectKind().GroupVersionKind().Kind, Namespace: r.GetNamespace(), Name: r.GetName(),
					Group: r.GroupVersionKind().Group, Version: r.GroupVersionKind().Version,
				},
				Action: string(libsveltosv1beta1.DeleteResourceAction),
			}
		}
	} else if canDelete(&r, currentPolicies) {
		logger.V(logs.LogVerbose).Info(fmt.Sprintf("remove owner reference %s/%s", r.GetNamespace(), r.GetName()))

		if isResourceOwner(&r, profile) {
			err := handleResourceDelete(ctx, c, &r, leavePolicies, logger)
			if err != nil {
				return nil, err
			}
		}
	}

	return resourceReport, nil
}

func isResourceOwner(resource *unstructured.Unstructured, profile client.Object) bool {
	// First consider annotations
	annotations := resource.GetAnnotations()
	if annotations != nil && annotations[OwnerKind] != "" {
		return annotations[OwnerKind] == profile.GetObjectKind().GroupVersionKind().Kind &&
			annotations[OwnerName] == profile.GetName()
	}

	if k8s_utils.IsOwnerReference(resource, profile) {
		return true
	}

	return false
}

// canDelete returns true if a policy can be deleted. For a policy to be deleted:
// - policy is not part of currentReferencedPolicies
func canDelete(policy client.Object, currentReferencedPolicies map[string]libsveltosv1beta1.Resource) bool {
	name := GetPolicyInfo(&libsveltosv1beta1.Resource{
		Kind:      policy.GetObjectKind().GroupVersionKind().Kind,
		Group:     policy.GetObjectKind().GroupVersionKind().Group,
		Version:   policy.GetObjectKind().GroupVersionKind().Version,
		Name:      policy.GetName(),
		Namespace: policy.GetNamespace(),
	})
	if _, ok := currentReferencedPolicies[name]; ok {
		return false
	}

	return true
}

func handleResourceDelete(ctx context.Context, c client.Client, policy client.Object,
	leavePolicies bool, logger logr.Logger) error {

	// If mode is set to LeavePolicies, leave policies in the workload cluster.
	// Remove all labels added by Sveltos.
	if leavePolicies {
		l := policy.GetLabels()
		delete(l, ReferenceKindLabel)
		delete(l, ReferenceNameLabel)
		delete(l, ReferenceNamespaceLabel)
		policy.SetLabels(l)

		annotations := policy.GetAnnotations()
		delete(annotations, OwnerKind)
		delete(annotations, OwnerName)
		delete(annotations, OwnerTier)
		policy.SetAnnotations(annotations)

		return c.Update(ctx, policy)
	}

	logger.V(logs.LogDebug).Info(fmt.Sprintf("removing resource %s %s/%s",
		policy.GetObjectKind().GroupVersionKind().Kind, policy.GetNamespace(), policy.GetName()))
	return c.Delete(ctx, policy)
}

// hasLabel search if key is one of the label.
// If value is empty, returns true if key is present.
// If value is not empty, returns true if key is present and value is a match.
func hasLabel(u *unstructured.Unstructured, key, value string) bool {
	lbls := u.GetLabels()
	if lbls == nil {
		return false
	}

	v, ok := lbls[key]

	if value == "" {
		return ok
	}

	return v == value
}

// hasAnnotation search if key is one of the annotation.
// If value is empty, returns true if key is present.
// If value is not empty, returns true if key is present and value is a match.
func hasAnnotation(u *unstructured.Unstructured, key, value string) bool {
	annotations := u.GetAnnotations()
	if annotations == nil {
		return false
	}

	v, ok := annotations[key]

	if value == "" {
		return ok
	}

	return v == value
}

func GetPolicyInfo(policy *libsveltosv1beta1.Resource) string {
	return fmt.Sprintf("%s.%s:%s:%s",
		policy.Kind,
		policy.Group,
		policy.Namespace,
		policy.Name)
}
