package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/enrichman/gocoverkube/internal/cli"

	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if k, found := os.LookupEnv("KUBECONFIG"); found {
		kubeconfig = k
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	rootCmd := cli.NewRootCmd(clientset, config)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
