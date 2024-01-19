package cmd

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"

	v1 "k8s.io/api/core/v1"
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
func Collect(ctx context.Context, clientset kubernetes.Interface, config *rest.Config, namespace, deploymentName string) error {
	// TODO add timeout flag

	deploymentClient := clientset.AppsV1().Deployments(namespace)
	deployment, err := deploymentClient.Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return err
	}

	podClient := clientset.CoreV1().Pods(namespace)
	pods, err := podClient.List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}

	oldPods := map[string]struct{}{}
	for _, p := range pods.Items {
		oldPods[p.Name] = struct{}{}
	}

	// restart deployment
	objectMeta := deployment.Spec.Template.ObjectMeta
	if objectMeta.Annotations == nil {
		objectMeta.Annotations = make(map[string]string)
	}
	objectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	deployment.Spec.Template.ObjectMeta = objectMeta

	_, err = deploymentClient.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner

	s.Suffix = " Restarting Deployment"
	s.Start()

	start := time.Now()
	for {
		pods, err := podClient.List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return err
		}

		oldPodFound := false
		for _, p := range pods.Items {
			if _, found := oldPods[p.Name]; found {
				oldPodFound = true
				break
			}
		}

		if !oldPodFound {
			break
		}

		time.Sleep(time.Second)
	}
	s.Stop()

	fmt.Printf("✅ Deployment restarted [%v]\n", time.Since(start).Round(time.Second))

	// TODO check for PVC before creating the pod

	_, err = podClient.Create(ctx, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: collectorName,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:    collectorName,
				Image:   "debian:stable-slim",
				Command: []string{"/bin/bash", "-c", "--"},
				Args:    []string{"while true; do sleep 30; done;"},
				VolumeMounts: []v1.VolumeMount{{
					Name:      volumeName,
					MountPath: mountPath,
				}},
			}},
			Volumes: []v1.Volume{{
				Name: volumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			}},
		},
	}, metav1.CreateOptions{})

	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
		fmt.Println(err)
	}

	s.Suffix = " Initializing collector"
	s.Start()

	start = time.Now()
	for {
		pod, err := podClient.Get(ctx, collectorName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if pod.Status.Phase == v1.PodRunning {
			break
		}

		time.Sleep(time.Second)
	}

	s.Stop()
	fmt.Printf("✅ Collector ready [%v]\n", time.Since(start).Round(time.Second))

	podExec := NewPodExec(config, clientset)
	// TODO: var destination
	_, out, _, err := podExec.PodCopyFile(collectorName+":/tmp/coverage", "coverage", namespace)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	fmt.Println("out:")
	fmt.Printf("%s", out.String())

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

func (p *PodExec) PodCopyFile(src string, dst string, namespace string) (*bytes.Buffer, *bytes.Buffer, *bytes.Buffer, error) {
	ioStreams, in, out, errOut := genericclioptions.NewTestIOStreams()
	copyOptions := cp.NewCopyOptions(ioStreams)

	copyOptions.Clientset = p.Clientset
	copyOptions.ClientConfig = p.RestConfig
	copyOptions.Container = collectorName
	copyOptions.Namespace = namespace

	err := copyOptions.Run([]string{src, dst})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not run copy operation: %v", err)
	}

	return in, out, errOut, nil
}
