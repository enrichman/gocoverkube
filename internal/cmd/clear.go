package cmd

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// gocoverkube clear
func ClearPod(ctx context.Context, clientset kubernetes.Interface, namespace, podName string) error {
	podClient := clientset.CoreV1().Pods(namespace)
	pod, err := podClient.Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	pod.Spec = clearPodSpec(ctx, pod.Spec)
	err = deleteAndCreatePod(ctx, clientset, namespace, pod)
	if err != nil {
		return err
	}

	err = deleteCollectorPod(ctx, clientset, namespace)
	if err != nil {
		return err
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(namespace)
	err = pvcClient.Delete(ctx, pvcName, metav1.DeleteOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	fmt.Println("✅ PVC deleted")

	return nil
}

// gocoverkube clear
func ClearDeployment(ctx context.Context, clientset kubernetes.Interface, namespace, deploymentName string) error {
	deploymentClient := clientset.AppsV1().Deployments(namespace)
	deployment, err := deploymentClient.Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	deployment.Spec.Template.Spec = clearPodSpec(ctx, deployment.Spec.Template.Spec)
	err = updateAndRestartDeployment(ctx, clientset, namespace, deployment)
	if err != nil {
		return err
	}

	err = deleteCollectorPod(ctx, clientset, namespace)
	if err != nil {
		return err
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(namespace)
	err = pvcClient.Delete(ctx, pvcName, metav1.DeleteOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	fmt.Println("✅ PVC deleted")

	return nil
}

func clearPodSpec(ctx context.Context, podSpec v1.PodSpec) v1.PodSpec {
	container := podSpec.Containers[0]
	// unset GOCOVERDIR env var
	container.Env = unsetEnvVar(container.Env)
	// unmount /tmp/coverage volume
	container.VolumeMounts = unsetVolumeMount(container.VolumeMounts)
	podSpec.Containers[0] = container

	// unbind /tmp/coverage volume to PVC
	podSpec.Volumes = unsetVolume(podSpec.Volumes)

	return podSpec
}

func unsetEnvVar(envVars []v1.EnvVar) []v1.EnvVar {
	originalEnvVars := []v1.EnvVar{}

	for _, e := range envVars {
		if e.Name != "GOCOVERDIR" {
			originalEnvVars = append(originalEnvVars, e)
		}
	}

	return originalEnvVars
}

func unsetVolumeMount(volumeMounts []v1.VolumeMount) []v1.VolumeMount {
	originalVolumeMounts := []v1.VolumeMount{}

	for _, vm := range volumeMounts {
		if vm.Name != volumeName {
			originalVolumeMounts = append(originalVolumeMounts, vm)
		}
	}

	return originalVolumeMounts
}

func unsetVolume(volumes []v1.Volume) []v1.Volume {
	originalVolumes := []v1.Volume{}

	for _, v := range volumes {
		if v.Name != volumeName {
			originalVolumes = append(originalVolumes, v)
		}
	}

	return originalVolumes
}
