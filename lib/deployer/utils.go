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
	"strconv"
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
	// ReferenceKindLabel is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the Kind (ConfigMap or Secret)
	// containing the policy.
	//
	// Deprecated: replaced by annotation
	ReferenceKindLabel = "projectsveltos.io/reference-kind"

	// ReferenceNameLabel is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the name of the ConfigMap/Secret
	// containing the policy.
	//
	// Deprecated: replaced by annotation
	ReferenceNameLabel = "projectsveltos.io/reference-name"

	// ReferenceNamespaceLabel is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the namespace of the ConfigMap/Secret
	// containing the policy.
	//
	// Deprecated: replaced by annotation
	ReferenceNamespaceLabel = "projectsveltos.io/reference-namespace"

	// ReferenceKindAnnotation is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the Kind (ConfigMap or Secret)
	// containing the policy.
	ReferenceKindAnnotation = "projectsveltos.io/reference-kind"

	// ReferenceNameAnnotation is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the name of the ConfigMap/Secret
	// containing the policy.
	ReferenceNameAnnotation = "projectsveltos.io/reference-name"

	// ReferenceNamespaceAnnotation is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the namespace of the ConfigMap/Secret
	// containing the policy.
	ReferenceNamespaceAnnotation = "projectsveltos.io/reference-namespace"

	// ReferenceTierAnnotation is added to each policy deployed by a ClusterSummary
	// instance to a managed Cluster. Indicates the namespace of the ConfigMap/Secret
	// containing the policy.
	ReferenceTierAnnotation = "projectsveltos.io/reference-tier"

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
	referenceTier int32, profile client.Object) (*ResourceInfo, error) {

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

	kind, namespace, name, tier := getReferenceInfo(currentObject)

	// if proposed tier is lower than current tier, do not check for conflict on referenced resource.
	if referenceTier >= tier {
		if kind != "" {
			if kind != referenceKind {
				return resourceInfo, &ConflictError{
					message: fmt.Sprintf("A conflict was detected while deploying resource %s:%s/%s. %s"+
						"This resource is currently deployed because of %s %s/%s.\n",
						object.GroupVersionKind().Kind, object.GetNamespace(), object.GetName(),
						getOwnerMessage(currentObject), kind, namespace, name)}
			}

			if namespace != referenceNamespace {
				return resourceInfo, &ConflictError{
					message: fmt.Sprintf("A conflict was detected while deploying resource %s:%s/%s. %s"+
						"This resource is currently deployed because of %s %s/%s.\n",
						object.GroupVersionKind().Kind, object.GetNamespace(), object.GetName(),
						getOwnerMessage(currentObject), kind, namespace, name)}
			}

			if name != referenceName {
				return resourceInfo, &ConflictError{
					message: fmt.Sprintf("A conflict was detected while deploying resource %s:%s/%s. %s"+
						"This resource is currently deployed because of %s %s/%s.\n",
						object.GroupVersionKind().Kind, object.GetNamespace(), object.GetName(),
						getOwnerMessage(currentObject), kind, namespace, name)}
			}
		}
	}

	err = validateSveltosOwner(object, currentObject, profile, kind, namespace, name)
	if err != nil {
		return resourceInfo, &ConflictError{message: err.Error()}
	}

	// Only in case object exists and there are no conflicts, return hash
	if annotations := currentObject.GetAnnotations(); annotations != nil {
		resourceInfo.Hash = annotations[PolicyHash]
	}

	return resourceInfo, nil
}

// getReferenceInfo extracts the kind, namespace, and name from object annotations
// and falls back to labels if the annotations are missing or the annotation map is nil.
func getReferenceInfo(object *unstructured.Unstructured) (kind, namespace, name string, tier int32) {
	// 1. Attempt to get info from Annotations
	annotations := object.GetAnnotations()
	if annotations != nil {
		kind = annotations[ReferenceKindAnnotation]
		namespace = annotations[ReferenceNamespaceAnnotation]
		name = annotations[ReferenceNameAnnotation]

		const defaultTier = 100
		tier := int32(defaultTier)
		tierStr, tierOk := annotations[ReferenceTierAnnotation]
		if tierOk {
			tier64, err := strconv.ParseInt(tierStr, 10, 32)
			if err == nil {
				tier = int32(tier64)
			}
			// If ParseInt fails, 'tier' remains the DefaultTier (100)
		}

		// If we found the kind, we assume the annotation set is complete enough.
		if kind != "" {
			return kind, namespace, name, tier
		}
	}

	// 2. Fallback to Labels if annotations were nil OR kind was not found in annotations
	labels := object.GetLabels()
	if labels != nil {
		var kindOk bool

		// NOTE: You had a bug in your original prompt's fallback logic where it
		// still referenced 'annotations'. This is corrected here to use 'labels'.
		kind, kindOk = labels[ReferenceKindLabel]
		namespace = labels[ReferenceNamespaceLabel]
		name = labels[ReferenceNameLabel]

		if kindOk {
			return kind, namespace, name, tier
		}
	}

	// 3. Info not found
	return "", "", "", tier
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
			return &ConflictError{
				message: fmt.Sprintf("A conflict was detected while deploying resource %s:%s/%s. %s"+
					"This resource is currently deployed because of %s %s/%s.\n",
					object.GroupVersionKind().Kind, object.GetNamespace(), object.GetName(),
					getOwnerMessage(currentObject), kind, namespace, name)}
		}
	} else if k8s_utils.HasSveltosResourcesAsOwnerReference(currentObject) && !k8s_utils.IsOwnerReference(currentObject, profile) {
		return &ConflictError{
			message: fmt.Sprintf("A conflict was detected while deploying resource %s:%s/%s. %s"+
				"This resource is currently deployed because of %s %s/%s.\n",
				object.GroupVersionKind().Kind, object.GetNamespace(), object.GetName(),
				getOwnerMessage(currentObject), kind, namespace, name)}
	}

	return nil
}

// getDetailedConflictMessage returns a message reporting the detected conflict and
// listing why this object is deployed. The message lists:
// - which is currently causing it to be deployed (owner)
// - which Secret/ConfigMap contains it
func getDetailedConflictMessage(ctx context.Context, dr dynamic.ResourceInterface,
	objectName string) (string, error) {

	currentObject, err := dr.Get(ctx, objectName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}

	var message string

	var kind, namespace, name string
	if annotations := currentObject.GetAnnotations(); annotations != nil {
		kind = annotations[ReferenceKindAnnotation]
		namespace = annotations[ReferenceNamespaceAnnotation]
		name = annotations[ReferenceNameAnnotation]
	}
	if kind == "" {
		// 2. Fallback to Labels if annotations were nil OR kind was not found in annotations
		if labels := currentObject.GetLabels(); labels != nil {
			kind = labels[ReferenceKindLabel]
			namespace = labels[ReferenceNamespaceLabel]
			name = labels[ReferenceNameLabel]
		}
	}

	ownerMessage := getOwnerMessage(currentObject)

	message += fmt.Sprintf("A conflict was detected while deploying resource %s:%s/%s. %s"+
		"This resource is currently deployed because of %s %s/%s.\n",
		currentObject.GroupVersionKind().Kind, currentObject.GetNamespace(), currentObject.GetName(),
		ownerMessage, kind, namespace, name)

	return message, nil
}

func getOwnerMessage(u *unstructured.Unstructured) string {
	message := "The Sveltos profile currently deploying this resource is "

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
