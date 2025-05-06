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
	"context"
	"fmt"
	"html/template"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectsveltos/libsveltos/lib/clusterproxy"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

// GetReferenceResourceNamespace determines the namespace for a referenced resource.
// If no namespace is provided, the cluster's namespace is used by default.
// If a namespace is provided, it may be a Go template string that can reference any field of the cluster object.
func GetReferenceResourceNamespace(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, referencedResourceNamespace string, clusterType libsveltosv1beta1.ClusterType,
) (string, error) {

	if referencedResourceNamespace == "" {
		return clusterNamespace, nil
	}

	return renderClusterTemplate(ctx, c, clusterNamespace, clusterName, referencedResourceNamespace, clusterType)
}

// `referencedResourceName` supports templating and can dynamically reference fields from the target cluster object.
// This allows the name of the referenced resource to be constructed based on any attribute of the cluster,
// including its namespace, name, and other metadata.
func GetReferenceResourceName(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, referencedResourceName string, clusterType libsveltosv1beta1.ClusterType,
) (string, error) {

	return renderClusterTemplate(ctx, c, clusterNamespace, clusterName, referencedResourceName, clusterType)
}

func getTemplateName(clusterNamespace, clusterName, requestorName string) string {
	return fmt.Sprintf("%s-%s-%s", clusterNamespace, clusterName, requestorName)
}

func renderClusterTemplate(ctx context.Context, c client.Client,
	clusterNamespace, clusterName, rawTemplate string, clusterType libsveltosv1beta1.ClusterType,
) (string, error) {

	if rawTemplate == "" {
		return clusterNamespace, nil
	}

	cluster, err := clusterproxy.GetCluster(ctx, c, clusterNamespace, clusterName, clusterType)
	if err != nil {
		return "", err
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
	if err != nil {
		return "", err
	}

	templateName := getTemplateName(clusterNamespace, clusterName, string(clusterType))

	tmpl, err := template.New(templateName).Option("missingkey=error").Funcs(ExtraFuncMap()).Parse(rawTemplate)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, struct {
		Cluster                       map[string]interface{}
		ClusterNamespace, ClusterName string
	}{
		Cluster:          u,
		ClusterNamespace: clusterNamespace,
		ClusterName:      clusterName,
	})
	if err != nil {
		return "", errors.Wrapf(err, "error executing template")
	}

	return buffer.String(), nil
}
