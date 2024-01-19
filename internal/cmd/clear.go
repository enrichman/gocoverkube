package cmd

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// gocoverkube clear
func Clear(ctx context.Context, clientset kubernetes.Interface, namespace, deploymentName string) error {
	deploymentClient := clientset.AppsV1().Deployments(namespace)
	deployment, err := deploymentClient.Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	deployment = clearDeployment(ctx, deployment)
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

func clearDeployment(ctx context.Context, deployment *appsv1.Deployment) *appsv1.Deployment {
	// update deployment
	podSpec := deployment.Spec.Template.Spec

	container := podSpec.Containers[0]
	// unset GOCOVERDIR env var
	container.Env = unsetEnvVar(container.Env)
	// unmount /tmp/coverage volume
	container.VolumeMounts = unsetVolumeMount(container.VolumeMounts)
	podSpec.Containers[0] = container

	// unbind /tmp/coverage volume to PVC
	podSpec.Volumes = unsetVolume(podSpec.Volumes)
	deployment.Spec.Template.Spec = podSpec

	return deployment
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
