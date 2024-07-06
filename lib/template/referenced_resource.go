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

package template

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GetReferenceResourceNamespace returns the namespace to use for a referenced resource.
// If namespace is set on referencedResource, that namespace will be used.
// If namespace is not set, cluster namespace will be used
func GetReferenceResourceNamespace(clusterNamespace, referencedResourceNamespace string) string {
	if referencedResourceNamespace != "" {
		return referencedResourceNamespace
	}

	return clusterNamespace
}

// Resources referenced can have their name expressed in function of cluster information:
// clusterNamespace => .Cluster.metadata.namespace
// clusterName => .Cluster.metadata.name
// clusterType => .Cluster.kind
//
// referencedResourceName can be expressed as a template using above cluster info
func GetReferenceResourceName(clusterNamespace, clusterName, clusterKind, referencedResourceName string) (string, error) {
	// Accept name that are templates
	templateName := getTemplateName(clusterNamespace, clusterName, clusterKind)

	tmpl, err := template.New(templateName).Option("missingkey=error").Funcs(ExtraFuncMap()).Parse(referencedResourceName)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer

	// Cluster namespace and name can be used to instantiate the name of the resource that
	// needs to be fetched from the management cluster. Defined an unstructured with namespace and name set
	u := &unstructured.Unstructured{}
	u.SetNamespace(clusterNamespace)
	u.SetName(clusterName)
	u.SetKind(clusterKind)

	if err := tmpl.Execute(&buffer,
		struct {
			Cluster map[string]interface{}
			// deprecated. This used to be original format which was different than rest of templating
			ClusterNamespace, ClusterName string
		}{
			Cluster:          u.UnstructuredContent(),
			ClusterNamespace: clusterNamespace,
			ClusterName:      clusterName}); err != nil {
		return "", errors.Wrapf(err, "error executing template")
	}
	return buffer.String(), nil
}

func getTemplateName(clusterNamespace, clusterName, requestorName string) string {
	return fmt.Sprintf("%s-%s-%s", clusterNamespace, clusterName, requestorName)
}
