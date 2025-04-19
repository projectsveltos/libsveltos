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

package deployer

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
)

const (
	// ReferenceLabelKind is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the Kind (ConfigMap or Secret)
	// containing the policy.
	ReferenceKindLabel = "projectsveltos.io/reference-kind"

	// ReferenceNameLabel is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the name of the ConfigMap/Secret
	// containing the policy.
	ReferenceNameLabel = "projectsveltos.io/reference-name"

	// ReferenceNamespaceLabel is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the namespace of the ConfigMap/Secret
	// containing the policy.
	ReferenceNamespaceLabel = "projectsveltos.io/reference-namespace"

	// PolicyHash is the annotation set on a policy when deployed in a managed
	// cluster.
	PolicyHash = "projectsveltos.io/hash"

	// OwnerTier is the annotation set on a policy when deployed in a managed
	// cluster. Contains the tier of the profile instance that deployed it.
	OwnerTier = "projectsveltos.io/owner-tier"

	// OwnerName is the annotation set on a policy when deployed in a managed
	// cluster. Contains the name of the profile instance that deployed it.
	OwnerName = "projectsveltos.io/owner-name"

	// OwnerKind is the annotation set on a policy when deployed in a managed
	// cluster. Contains the Kind of the profile instance that deployed it.
	OwnerKind = "projectsveltos.io/owner-kind"
)

type ConflictError struct {
	message string
}

func NewConflictError(msg string) *ConflictError {
	return &ConflictError{message: msg}
}

func (e *ConflictError) Error() string {
	return e.message
}

type ResourceInfo struct {
	CurrentResource *unstructured.Unstructured

	// Current profile owner's tier
	OwnerTier string

	Hash string
}

func (r *ResourceInfo) GetOwnerReferences() []corev1.ObjectReference {
	if r == nil {
		return nil
	}
	if r.CurrentResource == nil {
		return nil
	}

	references := r.CurrentResource.GetOwnerReferences()
	result := make([]corev1.ObjectReference, len(references))
	for i := range references {
		result[i] = corev1.ObjectReference{
			Kind:       references[i].Kind,
			APIVersion: references[i].APIVersion,
			Name:       references[i].Name,
		}
	}

	return result
}

func (r *ResourceInfo) GetResourceVersion() string {
	if r == nil {
		return ""
	}
	if r.CurrentResource == nil {
		return ""
	}
	return r.CurrentResource.GetResourceVersion()
}

// validateObjectForUpdate finds if object currently exists. If object exists:
// - verifies this object was created by same referenced object (specified by
// referenceKind, referenceNamespace, referenceName);
// - verifies this object was deployed because of the same profile instance (specified
// by profile instance).
// Returns an error otherwise.
// This is needed to prevent misconfigurations. An example would be when different
// ConfigMaps are referenced by ClusterProfile(s) or RoleRequest(s) and contain same policy
// namespace/name (content might be different) and are about to be deployed in the same cluster;
// Return an error if validation fails. Return also whether the object currently exists or not.
// If object exists, return value of PolicyHash annotation.
func ValidateObjectForUpdate(ctx context.Context, dr dynamic.ResourceInterface,
	object *unstructured.Unstructured, referenceKind, referenceNamespace, referenceName string,
	profile client.Object) (*ResourceInfo, error) {

	if object == nil {
		return nil, nil
	}

	resourceInfo := &ResourceInfo{}
	currentObject, err := dr.Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resourceInfo = nil
			return resourceInfo, nil
		}
		return nil, err
	}

	resourceInfo.CurrentResource = currentObject

	// If currently set, get the Tier of current owner
	if annotations := currentObject.GetAnnotations(); annotations != nil {
		if tierString, ok := annotations[OwnerTier]; ok {
			resourceInfo.OwnerTier = tierString
		}
	}

	if labels := currentObject.GetLabels(); labels != nil {
		kind, kindOk := labels[ReferenceKindLabel]
		namespace, namespaceOk := labels[ReferenceNamespaceLabel]
		name, nameOk := labels[ReferenceNameLabel]

		if kindOk {
			if kind != referenceKind {
				return resourceInfo, &ConflictError{
					message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
						object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name,
						addOwnerMessage(currentObject))}
			}
		}
		if namespaceOk {
			if namespace != referenceNamespace {
				return resourceInfo, &ConflictError{
					message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
						object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name,
						addOwnerMessage(currentObject))}
			}
		}
		if nameOk {
			if name != referenceName {
				return resourceInfo, &ConflictError{
					message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
						object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name,
						addOwnerMessage(currentObject))}
			}
		}

		if nameOk {
			if name != referenceName {
				return resourceInfo, &ConflictError{
					message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
						object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name,
						addOwnerMessage(currentObject))}
			}
		}

		err := validateSveltosOwner(object, currentObject, profile, kind, namespace, name)
		if err != nil {
			return resourceInfo, &ConflictError{message: err.Error()}
		}
	}

	// Only in case object exists and there are no conflicts, return hash
	if annotations := currentObject.GetAnnotations(); annotations != nil {
		resourceInfo.Hash = annotations[PolicyHash]
	}

	return resourceInfo, nil
}

func validateSveltosOwner(object, currentObject *unstructured.Unstructured, profile client.Object,
	kind, namespace, name string) error {

	var ownerName, ownerKind string
	// If currently set, get the Tier of current owner
	if annotations := currentObject.GetAnnotations(); annotations != nil {
		ownerName = annotations[OwnerName]
		ownerKind = annotations[OwnerKind]
	}

	if ownerName != "" {
		if ownerName != profile.GetName() || ownerKind != profile.GetObjectKind().GroupVersionKind().Kind {
			currentOwner := fmt.Sprintf("%s:%s", ownerKind, ownerName)
			return &ConflictError{
				message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
					object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name, currentOwner)}
		}
	} else if k8s_utils.HasSveltosResourcesAsOwnerReference(currentObject) && !k8s_utils.IsOwnerReference(currentObject, profile) {
		return &ConflictError{
			message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
				object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name,
				addOwnerMessage(currentObject))}
	}

	return nil
}

// GetOwnerMessage returns a message listing why this object is deployed. The message lists:
// - which is currently causing it to be deployed (owner)
// - which Secret/ConfigMap contains it
func GetOwnerMessage(ctx context.Context, dr dynamic.ResourceInterface,
	objectName string) (string, error) {

	currentObject, err := dr.Get(ctx, objectName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}

	var message string

	if labels := currentObject.GetLabels(); labels != nil {
		kind := labels[ReferenceKindLabel]
		namespace := labels[ReferenceNamespaceLabel]
		name := labels[ReferenceNameLabel]

		message += fmt.Sprintf("Object %s:%s/%s currently deployed because of %s %s/%s.\n",
			currentObject.GroupVersionKind().Kind, currentObject.GetNamespace(), currentObject.GetName(),
			kind, namespace, name)
	}

	message += addOwnerMessage(currentObject)

	return message, nil
}

func addOwnerMessage(u *unstructured.Unstructured) string {
	message := " Sveltos instance currently deploying this resource: "

	// First, if available, use annotation
	annotations := u.GetAnnotations()
	if annotations != nil && annotations[OwnerKind] != "" {
		message += fmt.Sprintf("%s %s;", annotations[OwnerKind], annotations[OwnerName])
		message += "\n"
		return message
	}

	// Resort to OwnerReference only if annotations are not set
	// This was old way, till release v0.52.0
	ownerRefs := u.GetOwnerReferences()
	for i := range ownerRefs {
		or := &ownerRefs[i]
		// Only include Sveltos resources
		if strings.Contains(or.APIVersion, "projectsveltos.io") {
			message += fmt.Sprintf("%s %s;", or.Kind, or.Name)
		}
	}
	message += "\n"
	return message
}
