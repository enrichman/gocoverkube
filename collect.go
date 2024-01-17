package main

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// gocoverkube collect
func Collect(ctx context.Context, clientset kubernetes.Interface, namespace, deploymentName string) error {
	deploymentClient := clientset.AppsV1().Deployments(namespace)
	_, err := deploymentClient.Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// restart deployment

	podClient := clientset.CoreV1().Pods(namespace)
	podClient.Create(ctx, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gocoverkube-collector",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:    "gocoverkube-collector",
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

	return nil
}
