package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func updateAndRestartDeployment(
	ctx context.Context,
	clientset kubernetes.Interface,
	namespace string,
	deployment *appsv1.Deployment,
) error {
	deploymentClient := clientset.AppsV1().Deployments(namespace)

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

	// update 'restartedAt' annotation to force restart
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

	s.Suffix = " Updating Deployment"
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
	return nil
}
