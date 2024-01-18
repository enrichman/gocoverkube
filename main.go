package main

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lmittmann/tint"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/cp"
	"k8s.io/kubectl/pkg/scheme"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	pvcName    = "gocoverkube-pvc"
	volumeName = "gocoverkube-tmp-coverage"
	mountPath  = "/tmp/coverage"
)

func main() {
	logger := slog.New(tint.NewHandler(os.Stderr, nil))

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

	////////////

	podExec := NewPodExec(*config, clientset)
	_, out, _, err := podExec.PodCopyFile("gocoverkube-collector:/tmp/coverage", "coverage", "gocoverkube-collector")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	fmt.Println("out:")
	fmt.Printf("%s", out.String())

	////////////

	rootCmd := NewRootCmd(clientset)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger.Info("done")
}

func checkConnection(clientset kubernetes.Interface) error {
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	fmt.Println(version.Platform)

	return nil
}

type PodExec struct {
	RestConfig *rest.Config
	*kubernetes.Clientset
}

func NewPodExec(config rest.Config, clientset *kubernetes.Clientset) *PodExec {
	config.APIPath = "/api"                                   // Make sure we target /api and not just /
	config.GroupVersion = &schema.GroupVersion{Version: "v1"} // this targets the core api groups so the url path will be /api/v1
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	return &PodExec{
		RestConfig: &config,
		Clientset:  clientset,
	}
}

func (p *PodExec) PodCopyFile(src string, dst string, containername string) (*bytes.Buffer, *bytes.Buffer, *bytes.Buffer, error) {
	ioStreams, in, out, errOut := genericclioptions.NewTestIOStreams()
	copyOptions := cp.NewCopyOptions(ioStreams)
	copyOptions.Clientset = p.Clientset
	copyOptions.ClientConfig = p.RestConfig
	copyOptions.Container = containername
	copyOptions.Namespace = "foo"
	err := copyOptions.Run([]string{src, dst})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not run copy operation: %v", err)
	}
	return in, out, errOut, nil
}
