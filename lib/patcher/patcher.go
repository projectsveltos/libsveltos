/*
Copyright 2021 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Credit: https://github.com/fluxcd/helm-controller/blob/42fe4a39c184ee586ae59fb83fb6556f7e976219/internal/postrender/kustomize.go
*/

package patcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	uyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	sveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

var (
	kustomizeRenderMutex sync.Mutex
)

// Kustomize is a Helm post-render plugin that runs Kustomize.
type CustomPatchPostRenderer struct {
	// Patches is a list of patches to apply to the rendered manifests.
	Patches []sveltosv1beta1.Patch
}

func (k *CustomPatchPostRenderer) RunUnstructured(unstructuredObjs []*unstructured.Unstructured,
) ([]*unstructured.Unstructured, error) {

	result := make([]*unstructured.Unstructured, 0, len(unstructuredObjs))

	for _, obj := range unstructuredObjs {
		// Filter patches that match this object's target
		matchingPatches, err := k.getMatchingPatches(obj)
		if err != nil {
			return nil, err
		}

		// Filter out patches where the path doesn't exist
		applicablePatches := k.filterApplicablePatches(obj, matchingPatches)

		if len(applicablePatches) == 0 {
			// No applicable patches, keep object as-is
			result = append(result, obj)
			continue
		}

		// Apply patches
		patchedObj, err := k.applyPatchesToObject(obj, applicablePatches)
		if err != nil {
			return nil, err
		}

		result = append(result, patchedObj)
	}

	return result, nil
}

func (k *CustomPatchPostRenderer) Run(renderedManifests *bytes.Buffer) (modifiedManifests *bytes.Buffer, err error) {
	fs := filesys.MakeFsInMemory()
	cfg := kustomizetypes.Kustomization{}
	cfg.APIVersion = kustomizetypes.KustomizationVersion
	cfg.Kind = kustomizetypes.KustomizationKind

	// Add rendered Helm output as input resource to the Kustomization.
	const input = "helm-output.yaml"
	cfg.Resources = append(cfg.Resources, input)
	if err := writeFile(fs, input, renderedManifests); err != nil {
		return nil, err
	}

	// Add patches.
	for _, m := range k.Patches {
		if err := validatePatch(m); err != nil {
			return nil, err
		}
		cfg.Patches = append(cfg.Patches, kustomizetypes.Patch{
			Patch:  m.Patch,
			Target: adaptSelector(m.Target),
		})
	}

	// Write kustomization config to file.
	kustomization, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if err := writeToFile(fs, "kustomization.yaml", kustomization); err != nil {
		return nil, err
	}

	resMap, err := buildKustomization(fs, ".")
	if err != nil {
		return nil, err
	}
	yaml, err := resMap.AsYaml()
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(yaml), nil
}

// validatePatch catches SM patches that lack metadata.name before kustomize does,
// returning a clear error instead of kustomize's opaque "unable to parse SM or JSON patch" message.
// A strategic merge patch is identified by having apiVersion/kind at the top level; if it
// also has a target selector with a name pattern, metadata.name must be present in the patch
// body so kustomize can identify the resource. JSON patches (lists starting with '-') are exempt.
func validatePatch(p sveltosv1beta1.Patch) error {
	trimmed := strings.TrimSpace(p.Patch)
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "- ") {
		return nil // JSON patch, no metadata.name required
	}

	rn, err := kyaml.Parse(trimmed)
	if err != nil {
		return nil // not parseable as YAML; let kustomize report the error
	}

	meta, err := rn.GetMeta()
	if err != nil || meta.Kind == "" {
		return nil // not a Kubernetes resource shape; let kustomize handle it
	}

	if meta.Name == "" {
		return fmt.Errorf(
			"strategic merge patch for %s/%s requires metadata.name in the patch body; "+
				"use a JSON patch (- op: ...) to match resources by regex name selector",
			meta.APIVersion, meta.Kind)
	}

	return nil
}

// getMatchingPatches returns patches that match the given object
func (k *CustomPatchPostRenderer) getMatchingPatches(obj *unstructured.Unstructured,
) ([]sveltosv1beta1.Patch, error) {

	var matching []sveltosv1beta1.Patch

	for _, patch := range k.Patches {
		matches, err := patchMatchesObject(patch.Target, obj)
		if err != nil {
			return matching, err
		}
		if matches {
			matching = append(matching, patch)
		}
	}

	return matching, nil
}

// regexMatches checks if the pattern (target) matches the value (current value of the object field).
// If pattern is empty, it returns true (matches everything, including an empty value).
// If pattern is not a valid regular expression, it returns an error.
func regexMatches(pattern, value string) (bool, error) {
	if pattern == "" {
		return true, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	return re.MatchString(value), nil
}

// patchMatchesObject checks if a patch target matches the object
func patchMatchesObject(target *sveltosv1beta1.PatchSelector,
	obj *unstructured.Unstructured) (bool, error) {

	if target == nil {
		return true, nil
	}

	gvk := obj.GroupVersionKind()

	// Helper function to apply regex check and handle errors
	checkMatch := func(pattern, value, fieldName string) (bool, error) {
		matches, err := regexMatches(pattern, value)
		if err != nil {
			return false,
				fmt.Errorf("invalid regex for target field %s: %w", fieldName, err)
		}
		return matches, nil
	}

	var matches bool
	var err error

	// Check Group
	matches, err = checkMatch(target.Group, gvk.Group, "Group")
	if err != nil || !matches {
		return matches, err
	}

	// Check Version
	matches, err = checkMatch(target.Version, gvk.Version, "Version")
	if err != nil || !matches {
		return matches, err
	}

	// Check Kind
	matches, err = checkMatch(target.Kind, gvk.Kind, "Kind")
	if err != nil || !matches {
		return matches, err
	}

	// Check Namespace
	matches, err = checkMatch(target.Namespace, obj.GetNamespace(), "Namespace")
	if err != nil || !matches {
		return matches, err
	}

	// Check Name
	matches, err = checkMatch(target.Name, obj.GetName(), "Name")
	if err != nil || !matches {
		return matches, err
	}

	// Check LabelSelector
	if target.LabelSelector != "" {
		selector, err := labels.Parse(target.LabelSelector)
		if err != nil {
			// Invalid selector, skip this patch
			return false, err
		}

		objLabels := labels.Set(obj.GetLabels())
		if !selector.Matches(objLabels) {
			return false, nil
		}
	}

	// Check AnnotationSelector
	if target.AnnotationSelector != "" {
		selector, err := labels.Parse(target.AnnotationSelector)
		if err != nil {
			// Invalid selector, skip this patch
			return false, err
		}

		objAnnotations := labels.Set(obj.GetAnnotations())
		if !selector.Matches(objAnnotations) {
			return false, nil
		}
	}

	return true, nil
}

// filterApplicablePatches removes 'remove' operations from JSON patches where the target
// path does not exist in the object, preventing errors on no-op removals. SM patches are
// passed through unchanged. For multi-operation JSON patches, only the remove operations
// targeting missing paths are stripped; the rest are kept.
func (k *CustomPatchPostRenderer) filterApplicablePatches(obj *unstructured.Unstructured,
	patches []sveltosv1beta1.Patch) []sveltosv1beta1.Patch {

	var applicable []sveltosv1beta1.Patch

	for _, patch := range patches {
		filtered, keep := filterPatchOperations(patch, obj)
		if keep {
			applicable = append(applicable, filtered)
		}
	}

	return applicable
}

// filterPatchOperations filters out individual 'remove' operations where the JSON Pointer
// path does not exist in the object. Returns the (possibly modified) patch and whether it
// should be kept at all. SM patches are always kept unchanged.
func filterPatchOperations(patch sveltosv1beta1.Patch, obj *unstructured.Unstructured) (sveltosv1beta1.Patch, bool) {
	if !isJSONPatch(patch.Patch) {
		return patch, true
	}

	ops, err := parseJSONPatchOps(patch.Patch)
	if err != nil {
		return patch, true // unparseable; let kustomize report the error
	}

	var keepOps []map[string]interface{}
	for _, op := range ops {
		opStr, _ := op["op"].(string)
		pathStr, _ := op["path"].(string)
		if opStr == "remove" && pathStr != "" && !pathExistsInObject(obj, pathStr) {
			continue
		}
		keepOps = append(keepOps, op)
	}

	if len(keepOps) == 0 {
		return patch, false
	}

	if len(keepOps) == len(ops) {
		return patch, true
	}

	rebuilt, err := rebuildJSONPatch(keepOps)
	if err != nil {
		return patch, true
	}

	return sveltosv1beta1.Patch{Patch: rebuilt, Target: patch.Target}, true
}

// isJSONPatch reports whether a patch string is a JSON patch (RFC 6902), i.e. a YAML/JSON
// sequence of {op, path, ...} objects, as opposed to a strategic merge patch.
func isJSONPatch(patchStr string) bool {
	trimmed := strings.TrimSpace(patchStr)
	return strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "- ")
}

// parseJSONPatchOps parses a JSON patch (in YAML or JSON format) into a slice of operation maps.
func parseJSONPatchOps(patchStr string) ([]map[string]interface{}, error) {
	decoder := uyaml.NewYAMLToJSONDecoder(bytes.NewBufferString(patchStr))
	var ops []map[string]interface{}
	if err := decoder.Decode(&ops); err != nil {
		return nil, err
	}
	return ops, nil
}

// rebuildJSONPatch serializes a slice of operation maps back to a JSON patch string.
// JSON is used (rather than YAML) because it is valid YAML and avoids re-encoding issues.
func rebuildJSONPatch(ops []map[string]interface{}) (string, error) {
	data, err := json.Marshal(ops)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// decodeJSONPointerToken decodes a single RFC 6902 JSON Pointer token:
// ~1 → / and ~0 → ~ (in that order, per the spec).
func decodeJSONPointerToken(token string) string {
	token = strings.ReplaceAll(token, "~1", "/")
	token = strings.ReplaceAll(token, "~0", "~")
	return token
}

// pathExistsInObject checks whether the JSON Pointer path exists in the unstructured object.
// It decodes JSON Pointer escape sequences (~1 → /, ~0 → ~) and handles array segments.
func pathExistsInObject(obj *unstructured.Unstructured, jsonPath string) bool {
	keys := strings.Split(strings.Trim(jsonPath, "/"), "/")
	var current interface{} = obj.Object

	for _, key := range keys {
		if key == "" {
			continue
		}
		key = decodeJSONPointerToken(key)

		switch v := current.(type) {
		case map[string]interface{}:
			val, found := v[key]
			if !found {
				return false
			}
			current = val
		case []interface{}:
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(v) {
				return false
			}
			current = v[idx]
		default:
			return false
		}
	}

	return true
}

// applyPatchesToObject applies patches to a single object
func (k *CustomPatchPostRenderer) applyPatchesToObject(obj *unstructured.Unstructured, patches []sveltosv1beta1.Patch) (*unstructured.Unstructured, error) {
	var buf bytes.Buffer
	data, err := kyaml.Marshal(obj.Object)
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	tempRenderer := &CustomPatchPostRenderer{Patches: patches}
	patchedBuf, err := tempRenderer.Run(&buf)
	if err != nil {
		return nil, err
	}

	objs, err := parseYAMLToUnstructured(patchedBuf)
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return obj, nil
	}

	return objs[0], nil
}

func writeToFile(fs filesys.FileSystem, path string, content []byte) error {
	helmOutput, err := fs.Create(path)
	if err != nil {
		return err
	}
	if _, err = helmOutput.Write(content); err != nil {
		return err
	}
	return helmOutput.Close()
}

func writeFile(fs filesys.FileSystem, path string, content *bytes.Buffer) error {
	helmOutput, err := fs.Create(path)
	if err != nil {
		return err
	}
	if _, err = content.WriteTo(helmOutput); err != nil {
		return err
	}
	return helmOutput.Close()
}

func adaptSelector(selector *sveltosv1beta1.PatchSelector) (output *kustomizetypes.Selector) {
	if selector != nil {
		output = &kustomizetypes.Selector{}
		output.Group = selector.Group
		output.Kind = selector.Kind
		output.Version = selector.Version
		output.Name = selector.Name
		output.Namespace = selector.Namespace
		output.LabelSelector = selector.LabelSelector
		output.AnnotationSelector = selector.AnnotationSelector
	}
	return
}

func parseYAMLToUnstructured(yamlData *bytes.Buffer) ([]*unstructured.Unstructured, error) {
	decoder := uyaml.NewYAMLToJSONDecoder(yamlData)
	var objs []*unstructured.Unstructured
	for {
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		objs = append(objs, &unstructured.Unstructured{Object: obj})
	}
	return objs, nil
}

// buildKustomization wraps krusty.MakeKustomizer with the following settings:
// - load files from outside the kustomization.yaml root
// - disable plugins except for the builtin ones
func buildKustomization(fs filesys.FileSystem, dirPath string) (resmap.ResMap, error) {
	// Temporary workaround for concurrent map read and map write bug
	// https://github.com/kubernetes-sigs/kustomize/issues/3659
	kustomizeRenderMutex.Lock()
	defer kustomizeRenderMutex.Unlock()

	buildOptions := &krusty.Options{
		LoadRestrictions: kustomizetypes.LoadRestrictionsNone,
		PluginConfig:     kustomizetypes.DisabledPluginConfig(),
	}

	k := krusty.MakeKustomizer(buildOptions)
	return k.Run(fs, dirPath)
}
