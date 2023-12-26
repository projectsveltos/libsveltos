package utils

import (
    "fmt"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "libsveltos"
)

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const TemplateAnnotation = "projectsveltos.io/template"

func IsTemplate(obj metav1.Object) bool {
    annotations := obj.GetAnnotations()
    if annotations != nil {
        _, ok := annotations[TemplateAnnotation]
        return ok
    }
    return false
}

func main() {
    // Create a Kubernetes client
    config, _ := clientcmd.BuildConfigFromFlags("", "")
    clientset, _ := kubernetes.NewForConfig(config)

    // Get a ConfigMap by name
    cm, _ := clientset.CoreV1().ConfigMaps("default").Get("my-configmap", metav1.GetOptions{})

    // Check if the ConfigMap is a template
    isTemplate := libsveltos.IsTemplate(cm)

    fmt.Println("Is the ConfigMap a template?", isTemplate)
}
