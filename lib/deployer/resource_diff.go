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
	"fmt"
	"io"
	"os"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// evaluateResourceDiff evaluates and returns diff
func evaluateResourceDiff(from, to *unstructured.Unstructured) (string, error) {
	objectInfo := fmt.Sprintf("%s %s", from.GroupVersionKind().Kind, from.GetName())

	// Remove managedFields, status and the hash annotation added by Sveltos.
	from = omitManagedFields(from)
	from = omitGeneratation(from)
	from = omitStatus(from)
	from = omitHashAnnotation(from)

	to = omitManagedFields(to)
	to = omitGeneratation(to)
	to = omitStatus(to)
	to = omitHashAnnotation(to)

	fromTempFile, err := os.CreateTemp("", "from-temp-file-")
	if err != nil {
		return "", err
	}
	defer os.Remove(fromTempFile.Name()) // Clean up the file after use
	fromWriter := io.Writer(fromTempFile)
	err = printUnstructured(from, fromWriter)
	if err != nil {
		return "", err
	}

	toTempFile, err := os.CreateTemp("", "to-temp-file-")
	if err != nil {
		return "", err
	}
	defer os.Remove(toTempFile.Name()) // Clean up the file after use
	toWriter := io.Writer(toTempFile)
	err = printUnstructured(to, toWriter)
	if err != nil {
		return "", err
	}

	fromContent, err := os.ReadFile(fromTempFile.Name())
	if err != nil {
		return "", err
	}

	toContent, err := os.ReadFile(toTempFile.Name())
	if err != nil {
		return "", err
	}

	edits := myers.ComputeEdits(span.URIFromPath(objectInfo), string(fromContent), string(toContent))

	diff := fmt.Sprint(gotextdiff.ToUnified(fmt.Sprintf("deployed: %s", objectInfo),
		fmt.Sprintf("proposed: %s", objectInfo), string(fromContent), edits))

	return diff, nil
}

func omitManagedFields(u *unstructured.Unstructured) *unstructured.Unstructured {
	a, err := meta.Accessor(u)
	if err != nil {
		// The object is not a `metav1.Object`, ignore it.
		return u
	}
	a.SetManagedFields(nil)
	return u
}

func omitGeneratation(u *unstructured.Unstructured) *unstructured.Unstructured {
	a, err := meta.Accessor(u)
	if err != nil {
		// The object is not a `metav1.Object`, ignore it.
		return u
	}
	a.SetGeneration(0)
	return u
}

func omitStatus(u *unstructured.Unstructured) *unstructured.Unstructured {
	content := u.UnstructuredContent()
	if _, ok := content["status"]; ok {
		content["status"] = ""
	}
	u.SetUnstructuredContent(content)
	return u
}

func omitHashAnnotation(u *unstructured.Unstructured) *unstructured.Unstructured {
	annotations := u.GetAnnotations()
	if annotations != nil {
		delete(annotations, PolicyHash)
	}
	u.SetAnnotations(annotations)
	return u
}

// Print the object inside the writer w.
func printUnstructured(obj runtime.Object, w io.Writer) error {
	if obj == nil {
		return nil
	}
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
