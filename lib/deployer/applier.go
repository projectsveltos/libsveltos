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

// Those methods are common between addon-controller and sveltos-applier

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2/textlogger"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/k8s_utils"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
	"github.com/projectsveltos/libsveltos/lib/patcher"
)

const (
	ReasonLabel = "projectsveltos.io/reason"
)

// CreateNamespace creates a namespace if it does not exist already
// No action in DryRun mode.
func CreateNamespace(ctx context.Context, clusterClient client.Client,
	isDryRun bool, namespaceName string) error {

	// No-op in DryRun mode
	if isDryRun {
		return nil
	}

	if namespaceName == "" {
		return nil
	}

	currentNs := &corev1.Namespace{}
	if err := clusterClient.Get(ctx, client.ObjectKey{Name: namespaceName}, currentNs); err != nil {
		if apierrors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			return clusterClient.Create(ctx, ns)
		}
		return err
	}
	return nil
}

func AddMetadata(policy *unstructured.Unstructured, resourceVersion string, profile client.Object,
	extraLabels, extraAnnotations map[string]string) {

	// The canDeployResource function validates if objects can be deployed. It achieves this by
	// fetching the object from the managed cluster and using its metadata to detect and potentially
	// resolve conflicts. If a conflict is detected, and the decision favors current clusterSummary instance,
	// the policy is updated.
	// However, it's crucial to ensure that between the time canDeployResource runs and the policy update,
	// no other modifications occur to the object. This includes updates from other ClusterSummary instances.
	// To guarantee consistency, we leverage the object's resourceVersion obtained by canDeployResource when
	// fetching the object. Setting the resource version during policy update acts as an optimistic locking mechanism.
	// If the object has been modified since the canDeployResource call, setting the resource version will fail,
	// invalidating previous conflict call and preventing unintended overwrites.
	// This approach ensures that conflict resolution decisions made by canDeployResource remain valid during the policy update.
	policy.SetResourceVersion(resourceVersion)

	addExtraLabels(policy, extraLabels)
	addExtraAnnotations(policy, extraAnnotations)
}

// addExtraLabels adds ExtraLabels to policy.
// If policy already has a label with a key present in `ExtraLabels`, the value from `ExtraLabels` will
// override the existing value.
func addExtraLabels(policy *unstructured.Unstructured, extraLabels map[string]string) {
	if extraLabels == nil {
		return
	}

	if len(extraLabels) == 0 {
		return
	}

	lbls := policy.GetLabels()
	if lbls == nil {
		lbls = map[string]string{}
	}
	for k := range extraLabels {
		lbls[k] = extraLabels[k]
	}

	policy.SetLabels(lbls)
}

// addExtraAnnotations adds ExtraAnnotations to policy.
// If policy already has an annotation with a key present in `ExtraAnnotations`, the value from `ExtraAnnotations`
// will override the existing value.
func addExtraAnnotations(policy *unstructured.Unstructured, extraAnnotations map[string]string) {
	if extraAnnotations == nil {
		return
	}

	if len(extraAnnotations) == 0 {
		return
	}

	annotations := policy.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	for k := range extraAnnotations {
		annotations[k] = extraAnnotations[k]
	}

	policy.SetAnnotations(annotations)
}

func GetUnstructured(section []byte, logger logr.Logger) ([]*unstructured.Unstructured, error) {
	elements, err := CustomSplit(string(section))
	if err != nil {
		return nil, err
	}
	policies := make([]*unstructured.Unstructured, 0, len(elements))
	for i := range elements {
		policy, err := k8s_utils.GetUnstructured([]byte(elements[i]))
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to get policy from Data %.100s", elements[i]))
			return nil, err
		}

		if policy == nil {
			logger.Error(err, fmt.Sprintf("failed to get policy from Data %.100s", elements[i]))
			return nil, fmt.Errorf("failed to get policy from Data %.100s", elements[i])
		}

		policies = append(policies, policy)
	}

	return policies, nil
}

// removeCommentsAndEmptyLines removes any line containing just YAML comments
// and any empty lines
func removeCommentsAndEmptyLines(text string) string {
	commentLine := regexp.MustCompile(`(?m)^\s*#([^#].*?)$`)
	result := commentLine.ReplaceAllString(text, "")
	emptyLine := regexp.MustCompile(`(?m)^\s*$`)
	result = emptyLine.ReplaceAllString(result, "")
	return result
}

func CustomSplit(text string) ([]string, error) {
	section := removeCommentsAndEmptyLines(text)
	if section == "" {
		return nil, nil
	}

	result := []string{}

	// First split by document separators if they exist
	var documents []string
	if strings.Contains(text, "---") {
		const bufferSize = 4096
		dec := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(text)), bufferSize)
		for {
			var value interface{}
			err := dec.Decode(&value)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, err
			}
			if value == nil {
				continue
			}
			valueBytes, err := yaml.Marshal(value)
			if err != nil {
				return nil, err
			}
			if valueBytes != nil {
				documents = append(documents, string(valueBytes))
			}
		}
	} else {
		documents = []string{text}
	}

	// Process each document - check if it's a YAML array
	for _, doc := range documents {
		trimmedDoc := strings.TrimSpace(doc)
		if strings.HasPrefix(trimmedDoc, "-") {
			// Try to parse as YAML array
			var resources []string
			err := yaml.Unmarshal([]byte(doc), &resources)
			if err == nil && len(resources) > 0 {
				result = append(result, resources...)
			} else {
				// If array parsing fails, treat as single document
				result = append(result, doc)
			}
		} else {
			// Single document
			result = append(result, doc)
		}
	}

	return result, nil
}

// GetResource returns sveltos Resource and the resource hash
func GetResource(policy *unstructured.Unstructured, ignoreForConfigurationDrift bool,
	referencedObject *corev1.ObjectReference, profile client.Object, tier int32, featureID string,
	logger logr.Logger) (resource *libsveltosv1beta1.Resource, policyHash string) {

	resource = &libsveltosv1beta1.Resource{
		Name:                        policy.GetName(),
		Namespace:                   policy.GetNamespace(),
		Kind:                        policy.GetKind(),
		Group:                       policy.GetObjectKind().GroupVersionKind().Group,
		Version:                     policy.GetObjectKind().GroupVersionKind().Version,
		IgnoreForConfigurationDrift: ignoreForConfigurationDrift,
	}

	var err error
	policyHash, err = ComputePolicyHash(policy)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to compute policy hash %v", err))
		policyHash = ""
	}

	// Get policy hash of referenced policy
	AddLabel(policy, ReferenceKindLabel, referencedObject.GetObjectKind().GroupVersionKind().Kind)
	AddLabel(policy, ReferenceNameLabel, referencedObject.Name)
	AddLabel(policy, ReferenceNamespaceLabel, referencedObject.Namespace)
	AddLabel(policy, ReasonLabel, featureID)
	AddAnnotation(policy, PolicyHash, policyHash)
	AddAnnotation(policy, OwnerTier, fmt.Sprintf("%d", tier))
	AddAnnotation(policy, OwnerName, profile.GetName())
	AddAnnotation(policy, OwnerKind, profile.GetObjectKind().GroupVersionKind().Kind)

	return resource, policyHash
}

// ComputePolicyHash compute policy hash.
func ComputePolicyHash(policy *unstructured.Unstructured) (string, error) {
	logger := textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))

	if policy.GetAnnotations() != nil {
		// Exclude Sveltos hash annotation if present.
		annotations := policy.GetAnnotations()
		delete(annotations, PolicyHash)
		policy.SetAnnotations(annotations)
	}

	// Convert to ordered map structure
	orderedObj := normalizeObject(policy.Object)
	logger.V(logs.LogInfo).Info(fmt.Sprintf("MGIANLUC orderedObj %v", orderedObj))

	jsonBytes, err := json.Marshal(orderedObj)
	if err != nil {
		return "", err
	}

	logger.V(logs.LogInfo).Info(fmt.Sprintf("MGIANLUC policy %s", string(jsonBytes)))

	resourceHash := sha256.Sum256(jsonBytes)
	logger.V(logs.LogInfo).Info(fmt.Sprintf("MGIANLUC hash %x", resourceHash))
	return fmt.Sprintf("sha256:%x", resourceHash), nil
}

func normalizeObject(obj interface{}) interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Create ordered map
		orderedMap := make(map[string]interface{})

		// Get sorted keys
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Build ordered map
		for _, k := range keys {
			orderedMap[k] = normalizeObject(v[k])
		}
		return orderedMap

	case []interface{}:
		// Recursively normalize array elements
		normalized := make([]interface{}, len(v))
		for i, item := range v {
			normalized[i] = normalizeObject(item)
		}
		return normalized

	default:
		// Primitive types return as-is
		return v
	}
}

// AddLabel adds label to an object
func AddLabel(obj metav1.Object, labelKey, labelValue string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[labelKey] = labelValue
	obj.SetLabels(labels)
}

// AddAnnotation adds annotation to an object
func AddAnnotation(obj metav1.Object, annotationKey, annotationValue string) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[annotationKey] = annotationValue
	obj.SetAnnotations(annotations)
}

// CanDeployResource verifies whether resource can be deployed. Following checks are performed:
//
// - if resource is currently already deployed in the managed cluster, if owned by this (Cluster)Profile/referenced
// resource => it can be updated
// - if resource is currently already deployed in the managed cluster and owned by same (Cluster)Profile but different
// referenced resource => it cannot be updated
// - if resource is currently already deployed in the managed cluster but owned by different (Cluster)Profile
// => it can be updated only if current (Cluster)Profile tier is lower than profile currently deploying the resource
//
// If resource cannot be deployed, return a ConflictError.
// If any other error occurs while doing those verification, the error is returned
func CanDeployResource(ctx context.Context, dr dynamic.ResourceInterface, policy *unstructured.Unstructured,
	referencedObject *corev1.ObjectReference, profile client.Object, profileTier int32, logger logr.Logger,
) (resourceInfo *ResourceInfo, requeueOldOwner bool, err error) {

	l := logger.WithValues("resource",
		fmt.Sprintf("%s:%s/%s", referencedObject.Kind, referencedObject.Namespace, referencedObject.Name))
	resourceInfo, err = ValidateObjectForUpdate(ctx, dr, policy,
		referencedObject.Kind, referencedObject.Namespace, referencedObject.Name, profile)
	if err != nil {
		var conflictErr *ConflictError
		ok := errors.As(err, &conflictErr)
		if ok {
			// There is a conflict.
			if HasHigherOwnershipPriority(getTier(resourceInfo.OwnerTier), profileTier) {
				l.V(logs.LogDebug).Info("conflict detected but resource ownership can change")
				// Because of tier, ownership must change. Which also means current ClusterProfile/Profile
				// owning the resource must be requeued for reconciliation
				return resourceInfo, true, nil
			}
			l.V(logs.LogDebug).Info("conflict detected")
			// Conflict cannot be resolved in favor of the clustersummary being reconciled. So report the conflict
			// error
			return resourceInfo, false, conflictErr
		}
		return nil, false, err
	}

	// There was no conflict. Resource can be deployed.
	return resourceInfo, false, nil
}

func GenerateConflictResourceReport(ctx context.Context, dr dynamic.ResourceInterface,
	resource *libsveltosv1beta1.Resource) *libsveltosv1beta1.ResourceReport {

	conflictReport := &libsveltosv1beta1.ResourceReport{
		Resource: *resource,
		Action:   string(libsveltosv1beta1.ConflictResourceAction),
	}
	message, err := GetOwnerMessage(ctx, dr, resource.Name)
	if err == nil {
		conflictReport.Message = message
	}
	return conflictReport
}

func removeDriftExclusionsFields(ctx context.Context, dr dynamic.ResourceInterface, isDritfDetectionMode, isDryRun bool,
	driftExclusions []libsveltosv1beta1.DriftExclusion, object *unstructured.Unstructured) (bool, error) {

	// When operating in SyncModeContinuousWithDriftDetection mode and DriftExclusions are specified,
	// avoid resetting certain object fields if the object is being redeployed (i.e, object already exists)
	// For example, consider a Deployment with an Autoscaler. Since the Autoscaler manages the spec.replicas
	// field, Sveltos is requested to deploy the Deployment and spec.replicas is specified as a field to ignore during
	// configuration drift evaluation.
	// If Sveltos is redeploying the deployment (for instance deployment image tag was changed), Sveltos must not
	// override spec.replicas.
	if isDritfDetectionMode {
		if driftExclusions != nil {
			_, err := dr.Get(ctx, object.GetName(), metav1.GetOptions{})
			if err == nil {
				// Resource exist. We are in drift detection mode and with driftExclusions.
				// Remove fields in driftExclusions before applying an update
				return true, nil
			} else if apierrors.IsNotFound(err) {
				// Object does not exist. We can apply it as it is. Since the object does
				// not exist, nothing will be overridden
				return false, nil
			} else {
				return false, err
			}
		}
	}

	if isDryRun && driftExclusions != nil {
		// When evaluating diff in DryRun mode, exclude fields
		return true, nil
	}

	return false, nil
}

// UpdateResource creates or updates a resource in a Cluster.
// No action in DryRun mode.
func UpdateResource(ctx context.Context, dr dynamic.ResourceInterface, isDriftDetection, isDryRun bool,
	driftExclusions []libsveltosv1beta1.DriftExclusion, object *unstructured.Unstructured, subresources []string,
	logger logr.Logger) (*unstructured.Unstructured, error) {

	forceConflict := true
	options := metav1.PatchOptions{
		FieldManager: "application/apply-patch",
		Force:        &forceConflict,
	}

	// No-op in DryRun mode
	if isDryRun {
		// Set dryRun option. Still proceed further so diff can be properly evaluated
		options.DryRun = []string{metav1.DryRunAll}
	}

	l := logger.WithValues("resourceNamespace", object.GetNamespace(), "resourceName", object.GetName(),
		"resourceGVK", object.GetObjectKind().GroupVersionKind(), "subresources", subresources)
	l.V(logs.LogDebug).Info("deploying policy")

	removeFields, err := removeDriftExclusionsFields(ctx, dr, isDriftDetection, isDryRun, driftExclusions, object)
	if err != nil {
		return nil, err
	}

	if removeFields {
		patches := TransformDriftExclusionsToPatches(driftExclusions)
		p := &patcher.CustomPatchPostRenderer{Patches: patches}
		var patchedObjects []*unstructured.Unstructured
		patchedObjects, err = p.RunUnstructured([]*unstructured.Unstructured{object})
		if err != nil {
			return nil, err
		}
		object = patchedObjects[0]
	}

	var updatedObject *unstructured.Unstructured
	if isCustomResourceDefinition(object) {
		updatedObject, err = updateCRD(ctx, dr, isDryRun, object)
	} else {
		var data []byte
		data, err = runtime.Encode(unstructured.UnstructuredJSONScheme, object)
		if err != nil {
			return nil, err
		}
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var retryErr error
			updatedObject, retryErr = dr.Patch(ctx, object.GetName(), types.ApplyPatchType, data, options)
			if retryErr != nil {
				if isDryRun && apierrors.IsNotFound(retryErr) {
					// In DryRun mode, if resource is namespaced and namespace is not present,
					// patch will fail with namespace not found. Treat this error as resoruce
					// would be created
					updatedObject = object
					return nil
				}
				return retryErr
			}
			return nil
		})
	}
	if err != nil {
		return nil, err
	}

	return updatedObject, applySubresources(ctx, dr, object, subresources, &options)
}

func isCustomResourceDefinition(u *unstructured.Unstructured) bool {
	gvk := schema.FromAPIVersionAndKind(u.GetAPIVersion(), u.GetKind())

	if u.GetKind() == "CustomResourceDefinition" && gvk.Group == apiextensions.GroupName {
		return true
	}

	return false
}

func updateCRD(ctx context.Context, dr dynamic.ResourceInterface, isDryRun bool, u *unstructured.Unstructured,
) (*unstructured.Unstructured, error) {

	createOptions := metav1.CreateOptions{}
	if isDryRun {
		// Set dryRun option. Still proceed further so diff can be properly evaluated
		createOptions.DryRun = []string{metav1.DryRunAll}
	}

	var updatedObject *unstructured.Unstructured
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var retryErr error
		updatedObject, retryErr = dr.Get(ctx, u.GetName(), metav1.GetOptions{})
		if retryErr != nil {
			if apierrors.IsNotFound(retryErr) {
				updatedObject, retryErr = dr.Create(ctx, u, createOptions)
				return retryErr
			}
			return retryErr
		}

		updateOptions := metav1.UpdateOptions{}
		if isDryRun {
			// Set dryRun option. Still proceed further so diff can be properly evaluated
			updateOptions.DryRun = []string{metav1.DryRunAll}
		}

		u.SetResourceVersion(updatedObject.GetResourceVersion())
		updatedObject, retryErr = dr.Update(ctx, u, updateOptions)
		return retryErr
	})

	return updatedObject, err
}

// transformDriftExclusionPathsToPatches transforms a DriftExclusion instance to a Patch instance.
// Operation is always set to remove (the goal of a DriftExclusion is to not consider, so to remove, a path
// during configuration drift evaluation).
func transformDriftExclusionPathsToPatches(driftExclusion *libsveltosv1beta1.DriftExclusion) []libsveltosv1beta1.Patch {
	if len(driftExclusion.Paths) == 0 {
		return nil
	}

	patches := make([]libsveltosv1beta1.Patch, len(driftExclusion.Paths))
	for i := range driftExclusion.Paths {
		path := driftExclusion.Paths[i]
		// This patch is exclusively used for removing fields. The drift-detection-manager applies it upon detecting
		// changes to Sveltos-deployed resources. By removing the specified field, it prevents the field from being
		// considered during configuration drift evaluation.
		patches[i] = libsveltosv1beta1.Patch{
			Target: driftExclusion.Target,
			Patch: fmt.Sprintf(`- op: remove
  path: %s`, path),
		}
	}

	return patches
}

// TransformDriftExclusionsToPatches transforms a slice of driftExclusion to a slice of Patch
// Operation on each Patch is always set to remove (the goal of a DriftExclusion is to not consider, so to remove,
// a path during configuration drift evaluation).
func TransformDriftExclusionsToPatches(driftExclusions []libsveltosv1beta1.DriftExclusion) []libsveltosv1beta1.Patch {
	patches := []libsveltosv1beta1.Patch{}

	for i := range driftExclusions {
		item := &driftExclusions[i]
		tmpPatches := transformDriftExclusionPathsToPatches(item)
		patches = append(patches, tmpPatches...)
	}

	return patches
}

func applySubresources(ctx context.Context, dr dynamic.ResourceInterface,
	object *unstructured.Unstructured, subresources []string, options *metav1.PatchOptions) error {

	if len(subresources) == 0 {
		return nil
	}

	object.SetManagedFields(nil)
	object.SetResourceVersion("")
	data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, object)
	if err != nil {
		return err
	}

	_, err = dr.Patch(ctx, object.GetName(), types.ApplyPatchType, data, *options, subresources...)
	return err
}

func GenerateResourceReport(policyHash string, resourceInfo *ResourceInfo,
	policy *unstructured.Unstructured, resource *libsveltosv1beta1.Resource) *libsveltosv1beta1.ResourceReport {

	resourceReport := &libsveltosv1beta1.ResourceReport{Resource: *resource}
	if resourceInfo == nil {
		resourceReport.Action = string(libsveltosv1beta1.CreateResourceAction)
	} else if policyHash != resourceInfo.Hash {
		resourceReport.Action = string(libsveltosv1beta1.UpdateResourceAction)
		diff, err := evaluateResourceDiff(resourceInfo.CurrentResource, policy)
		if err == nil {
			resourceReport.Message = diff
		}
	} else {
		resourceReport.Action = string(libsveltosv1beta1.NoResourceAction)
		resourceReport.Message = "Object already deployed. And policy referenced by ClusterProfile has not changed since last deployment."
	}

	return resourceReport
}

func HandleDeployUnstructuredErrors(conflictErrorMsg, errorMsg string, isDryRun bool) error {
	if conflictErrorMsg != "" {
		if !isDryRun {
			// if in DryRun mode, ignore conflicts
			return NewConflictError(conflictErrorMsg)
		}
	}

	if errorMsg != "" {
		if !isDryRun {
			// if in DryRun mode, ignore errors
			return fmt.Errorf("%s", errorMsg)
		}
	}

	return nil
}
