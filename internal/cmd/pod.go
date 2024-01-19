package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func createCollectorPod(
	ctx context.Context,
	clientset kubernetes.Interface,
	namespace string,
) error {
	podClient := clientset.CoreV1().Pods(namespace)

	_, err := podClient.Create(ctx, &v1.Pod{
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
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Creating Collector Pod"
	s.Start()

	start := time.Now()

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
	fmt.Printf("✅ Collector Pod created [%v]\n", time.Since(start).Round(time.Second))

	return nil
}

func deleteCollectorPod(
	ctx context.Context,
	clientset kubernetes.Interface,
	namespace string,
) error {
	podClient := clientset.CoreV1().Pods(namespace)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Deleting Collector Pod"
	s.Start()

	start := time.Now()

	err := podClient.Delete(ctx, collectorName, metav1.DeleteOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	for {
		_, err = podClient.Get(ctx, collectorName, metav1.GetOptions{})
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}
			break
		}

		time.Sleep(time.Second)
	}

	s.Stop()

	fmt.Printf("✅ Collector Pod deleted [%v]\n", time.Since(start).Round(time.Second))

	return nil
}

func deleteAndCreatePod(
	ctx context.Context,
	clientset kubernetes.Interface,
	namespace string,
	pod *v1.Pod,
) error {
	podClient := clientset.CoreV1().Pods(namespace)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Deleting Pod"
	s.Start()

	start := time.Now()

	err := podClient.Delete(ctx, pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	for {
		_, err := podClient.Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			// pod was deleted
			if k8serrors.IsNotFound(err) {
				break
			}
			return err
		}

		time.Sleep(time.Second)
	}

	s.Stop()

	fmt.Printf("✅ Pod deleted [%v]\n", time.Since(start).Round(time.Second))

	s.Suffix = " Creating Pod"
	s.Start()

	start = time.Now()

	pod.ResourceVersion = ""

	_, err = podClient.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	for {
		pod, err := podClient.Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if pod.Status.Phase == v1.PodRunning {
			break
		}

		time.Sleep(time.Second)
	}

	s.Stop()

	fmt.Printf("✅ Pod created [%v]\n", time.Since(start).Round(time.Second))

	return nil
}
