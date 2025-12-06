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

// filterApplicablePatches removes patches where operation is 'remove' and the target path doesn't exist
func (k *CustomPatchPostRenderer) filterApplicablePatches(obj *unstructured.Unstructured,
	patches []sveltosv1beta1.Patch) []sveltosv1beta1.Patch {

	var applicable []sveltosv1beta1.Patch

	for _, patch := range patches {
		// Parse the patch to get operation and path
		op, path := extractOpAndPath(patch.Patch)

		// Only filter if operation is 'remove' and path doesn't exist
		if op == "remove" && path != "" && !pathExistsInObject(obj, path) {
			// Path doesn't exist, skip this remove operation
			continue
		}

		applicable = append(applicable, patch)
	}

	return applicable
}

// extractOpAndPath extracts the operation and path from a patch string
// Example: "- op: remove\n  path: /spec/template/spec/nodeSelector" -> ("remove", "/spec/template/spec/nodeSelector")
func extractOpAndPath(patchStr string) (op, path string) {
	// We only expect one YAML list item, so we can treat the whole string as one block of fields.
	// However, if the input is truly multiline, splitting is fine.
	lines := strings.Split(patchStr, "\n")

	// Combine all lines into a single, space-separated string for simpler parsing.
	// This allows us to search for "op:" and "path:" anywhere, regardless of line breaks.
	singleLinePatch := strings.Join(lines, " ")

	// --- Extraction Logic for 'op' ---
	opIndex := strings.Index(singleLinePatch, "op:")
	if opIndex != -1 {
		// Find the start of the value: "op:" is 3 characters long
		opValueString := singleLinePatch[opIndex+3:]

		// Find the end of the value. Assume the value ends at the start of the next key
		// (like "path:") or the end of the string.
		// We'll use TrimSpace for simplicity, as op values are usually single words.
		op = strings.TrimSpace(opValueString)

		// In case the path key follows immediately on the same line,
		// trim everything after "path:" or "Path:" from the 'op' value.
		// This handles cases like "remove path: /foo"
		if idx := strings.Index(op, "path:"); idx != -1 {
			op = op[:idx]
		}

		op = strings.TrimSpace(op)
		op = strings.Trim(op, `"'`)
	}

	// --- Extraction Logic for 'path' ---
	pathIndex := strings.Index(singleLinePatch, "path:")
	if pathIndex != -1 {
		// Find the start of the value: "path:" is 5 characters long
		pathValueString := singleLinePatch[pathIndex+5:]

		// Trim the value of path
		path = strings.TrimSpace(pathValueString)

		// If the 'op' key follows immediately on the same line, trim everything after "op:".
		// This handles cases like "/foo op: remove"
		if idx := strings.Index(path, "op:"); idx != -1 {
			path = path[:idx]
		}

		path = strings.TrimSpace(path)
		path = strings.Trim(path, `"'`)
	}

	return op, path
}

// pathExistsInObject checks if a JSON path exists in the unstructured object
func pathExistsInObject(obj *unstructured.Unstructured, jsonPath string) bool {
	keys := strings.Split(strings.Trim(jsonPath, "/"), "/")
	current := obj.Object

	for _, key := range keys {
		if key == "" {
			continue
		}

		val, found := current[key]
		if !found {
			return false
		}

		// Navigate deeper if it's a nested object
		if nested, ok := val.(map[string]interface{}); ok {
			current = nested
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
