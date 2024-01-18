package cmd

import (
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

func ServerVersion(clientset kubernetes.Interface) (*version.Info, error) {
	return clientset.Discovery().ServerVersion()
}
