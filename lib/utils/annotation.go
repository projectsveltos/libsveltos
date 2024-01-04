package utils

import (
    "fmt"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "github.com/projectsveltos/libsveltos/lib/utils"
)

// This annotation is added on referenced ConfigMap/Secret to indicate their content is a template

const TemplateAnnotation = "projectsveltos.io/template"

func IsTemplate(obj metav1.Object) bool {
    annotations := obj.GetAnnotations()
    if annotations != nil {
        _, ok := annotations[TemplateAnnotation]
        return ok
    }
    return false
}


