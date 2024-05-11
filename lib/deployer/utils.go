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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ReferenceLabelKind is added to each policy deployed by a ClusterSummary
	// instance to a CAPI Cluster. Indicates the Kind (ConfigMap or Secret)
	// containing the policy.
	ReferenceKindLabel = "projectsveltos.io/reference-kind"

	// ReferenceNameLabel is added to each policy deployed by a ClusterSummary
	// instance to a CAPI Cluster. Indicates the name of the ConfigMap/Secret
	// containing the policy.
	ReferenceNameLabel = "projectsveltos.io/reference-name"

	// ReferenceNamespaceLabel is added to each policy deployed by a ClusterSummary
	// instance to a CAPI Cluster. Indicates the namespace of the ConfigMap/Secret
	// containing the policy.
	ReferenceNamespaceLabel = "projectsveltos.io/reference-namespace"

	// PolicyHash is the annotation set on a policy when deployed in a CAPI
	// cluster.
	PolicyHash = "projectsveltos.io/hash"
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

// validateObjectForUpdate finds if object currently exists. If object exists:
// - verifies this object was created by same ConfigMap/Secret. Returns an error otherwise.
// This is needed to prevent misconfigurations. An example would be when different
// ConfigMaps are referenced by ClusterProfile(s) or RoleRequest(s) and contain same policy
// namespace/name (content might be different) and are about to be deployed in the same cluster;
// Return an error if validation fails. Return also whether the object currently exists or not.
// If object exists, return value of PolicyHash annotation.
func ValidateObjectForUpdate(ctx context.Context, dr dynamic.ResourceInterface,
	object *unstructured.Unstructured,
	referenceKind, referenceNamespace, referenceName string) (exist bool, hash string, err error) {

	if object == nil {
		return false, "", nil
	}

	currentObject, err := dr.Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, "", nil
		}
		return false, "", err
	}

	if labels := currentObject.GetLabels(); labels != nil {
		kind, kindOk := labels[ReferenceKindLabel]
		namespace, namespaceOk := labels[ReferenceNamespaceLabel]
		name, nameOk := labels[ReferenceNameLabel]

		if kindOk {
			if kind != referenceKind {
				return true, "", &ConflictError{
					message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
						object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name,
						addListOfOwners(currentObject))}
			}
		}
		if namespaceOk {
			if namespace != referenceNamespace {
				return true, "", &ConflictError{
					message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
						object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name,
						addListOfOwners(currentObject))}
			}
		}
		if nameOk {
			if name != referenceName {
				return true, "", &ConflictError{
					message: fmt.Sprintf("conflict: policy (kind: %s) %s/%s is currently deployed by %s: %s/%s.\n%s",
						object.GetKind(), object.GetNamespace(), object.GetName(), kind, namespace, name,
						addListOfOwners(currentObject))}
			}
		}
	}

	// Only in case object exists and there are no conflicts, return hash
	if annotations := currentObject.GetAnnotations(); annotations != nil {
		hash = annotations[PolicyHash]
	}

	return true, hash, nil
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

		message += fmt.Sprintf("Object currently deployed because of %s %s/%s.", kind, namespace, name)
	}

	message += addListOfOwners(currentObject)

	return message, nil
}

func addListOfOwners(u *unstructured.Unstructured) string {
	message := "List of conflicting ClusterProfiles/Profiles:"
	ownerRefs := u.GetOwnerReferences()
	for i := range ownerRefs {
		or := &ownerRefs[i]
		message += fmt.Sprintf("%s %s;", or.Kind, or.Name)
	}

	return message
}

// AddOwnerReference adds Sveltos resource owning a resource as an object's OwnerReference.
// OwnerReferences are used as ref count. Different Sveltos resources might match same cluster and
// reference same ConfigMap. This means a policy contained in a ConfigMap is deployed in a Cluster
// because of different Sveltos resources.
// When cleaning up, a policy can be removed only if no more Sveltos resources are listed as OwnerReferences.
func AddOwnerReference(object, owner client.Object) {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		onwerReferences = make([]metav1.OwnerReference, 0)
	}

	for i := range onwerReferences {
		ref := &onwerReferences[i]
		if ref.Kind == owner.GetObjectKind().GroupVersionKind().Kind &&
			ref.Name == owner.GetName() {

			return
		}
	}

	apiVersion, kind := owner.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()

	onwerReferences = append(onwerReferences,
		metav1.OwnerReference{
			APIVersion: apiVersion,
			Kind:       kind,
			Name:       owner.GetName(),
			UID:        owner.GetUID(),
		},
	)

	object.SetOwnerReferences(onwerReferences)
}

// RemoveOwnerReference removes Sveltos resource as an OwnerReference from object.
// OwnerReferences are used as ref count. Different Sveltos resources might match same cluster and
// reference same ConfigMap. This means a policy contained in a ConfigMap is deployed in a Cluster
// because of different SveltosResources. When cleaning up, a policy can be removed only if no more
// SveltosResources are listed as OwnerReferences.
func RemoveOwnerReference(object, owner client.Object) {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		return
	}

	_, kind := owner.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()

	for i := range onwerReferences {
		ref := &onwerReferences[i]
		if ref.Kind == kind &&
			ref.Name == owner.GetName() {

			onwerReferences[i] = onwerReferences[len(onwerReferences)-1]
			onwerReferences = onwerReferences[:len(onwerReferences)-1]
			break
		}
	}

	object.SetOwnerReferences(onwerReferences)
}

// IsOnlyOwnerReference returns true if clusterprofile is the only ownerreference for object
func IsOnlyOwnerReference(object, owner client.Object) bool {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		return false
	}

	if len(onwerReferences) != 1 {
		return false
	}

	kind := owner.GetObjectKind().GroupVersionKind().Kind

	ref := &onwerReferences[0]
	return ref.Kind == kind &&
		ref.Name == owner.GetName()
}

// IsOwnerReference returns true is owner is one of the OwnerReferences
// for object
func IsOwnerReference(object, owner client.Object) bool {
	onwerReferences := object.GetOwnerReferences()
	if onwerReferences == nil {
		return false
	}

	kind := owner.GetObjectKind().GroupVersionKind().Kind

	for i := range onwerReferences {
		ref := &onwerReferences[i]
		if ref.Kind == kind &&
			ref.Name == owner.GetName() {

			return true
		}
	}

	return false
}
