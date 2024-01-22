package cmd

import (
	"context"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/cp"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	collectorName = "gocoverkube-collector"
)

// gocoverkube collect
func Collect(ctx context.Context, clientset kubernetes.Interface, config *rest.Config, namespace, deploymentName, outDst string) error {
	// TODO add timeout flag

	deploymentClient := clientset.AppsV1().Deployments(namespace)
	deployment, err := deploymentClient.Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(namespace)
	_, err = pvcClient.Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return errors.New("PVC not found. Did you run 'init'?")
		}
		return err
	}

	err = createCollectorPod(ctx, clientset, namespace)
	if err != nil {
		return err
	}

	err = updateAndRestartDeployment(ctx, clientset, namespace, deployment)
	if err != nil {
		return err
	}

	podExec := NewPodExec(config, clientset)
	err = podExec.PodCopyFile(collectorName+":/tmp/coverage", outDst, namespace)
	if err != nil {
		return err
	}

	fmt.Printf("ℹ️  Coverage collected at '%s'\n", outDst)

	return nil
}

func CollectPod(ctx context.Context, clientset kubernetes.Interface, config *rest.Config, namespace, podName, outDst string) error {
	// TODO add timeout flag

	podClient := clientset.CoreV1().Pods(namespace)
	pod, err := podClient.Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(namespace)
	_, err = pvcClient.Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return errors.New("PVC not found. Did you run 'init'?")
		}
		return err
	}

	err = createCollectorPod(ctx, clientset, namespace)
	if err != nil {
		return err
	}

	err = deleteAndCreatePod(ctx, clientset, namespace, pod)
	if err != nil {
		return err
	}

	podExec := NewPodExec(config, clientset)
	err = podExec.PodCopyFile(collectorName+":/tmp/coverage", outDst, namespace)
	if err != nil {
		return err
	}

	fmt.Printf("ℹ️  Coverage collected at '%s'\n", outDst)

	return nil
}

type PodExec struct {
	RestConfig *rest.Config
	Clientset  kubernetes.Interface
}

func NewPodExec(config *rest.Config, clientset kubernetes.Interface) *PodExec {
	config.APIPath = "/api"                                   // Make sure we target /api and not just /
	config.GroupVersion = &schema.GroupVersion{Version: "v1"} // this targets the core api groups so the url path will be /api/v1
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	return &PodExec{
		RestConfig: config,
		Clientset:  clientset,
	}
}

func (p *PodExec) PodCopyFile(src string, dst string, namespace string) error {
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams()
	copyOptions := cp.NewCopyOptions(ioStreams)

	copyOptions.Clientset = p.Clientset
	copyOptions.ClientConfig = p.RestConfig
	copyOptions.Container = collectorName
	copyOptions.Namespace = namespace

	err := copyOptions.Run([]string{src, dst})
	if err != nil {
		return fmt.Errorf("Could not run copy operation: %v", err)
	}
	return nil
}
