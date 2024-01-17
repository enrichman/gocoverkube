package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lmittmann/tint"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	volumeName = "gocoverkube-tmp-coverage"
	mountPath  = "/tmp/coverage"
)

func main() {
	logger := slog.New(tint.NewHandler(os.Stderr, nil))

	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	err = Init(context.Background(), clientset, "foo", "debian-slim")
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("done")
}

// gocoverkube init
func Init(ctx context.Context, clientset kubernetes.Interface, namespace, deploymentName string) error {
	storageClass, err := getDefaultStorageClass(ctx, clientset)
	if err != nil {
		return err
	}

	deploymentClient := clientset.AppsV1().Deployments(namespace)
	deployment, err := deploymentClient.Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(namespace)
	pvc, err := claimPersistentVolume(ctx, pvcClient, storageClass)
	if err != nil {
		return err
	}

	err = patchDeployment(ctx, deploymentClient, deployment, pvc)
	if err != nil {
		return err
	}

	return nil
}

// getDefaultStorageClass will get the default storage class
func getDefaultStorageClass(ctx context.Context, clientset kubernetes.Interface) (string, error) {
	storageClassesClient := clientset.StorageV1().StorageClasses()

	storageClasses, err := storageClassesClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, sc := range storageClasses.Items {
		defaultClass := sc.Annotations["storageclass.kubernetes.io/is-default-class"]
		if defaultClass == "true" {
			return sc.Name, nil
		}
	}

	return "", errors.New("storage class not found")
}

// claimPersistentVolume
func claimPersistentVolume(ctx context.Context, pvcClient typedcorev1.PersistentVolumeClaimInterface, storageClass string) (string, error) {
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gocoverkube",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gocoverkube",
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			StorageClassName: &storageClass,
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("100M"),
				},
			},
		},
	}

	_, err := pvcClient.Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return "", err
		}
		fmt.Println(err)
	}

	return pvc.Name, nil
}

func patchDeployment(ctx context.Context, deploymentClient typedappsv1.DeploymentInterface, deployment *appsv1.Deployment, pvc string) error {
	// update deployment
	podSpec := deployment.Spec.Template.Spec

	container := podSpec.Containers[0]
	// add GOCOVERDIR env var
	container.Env = setEnvVar(container.Env)
	// mount /tmp/coverage volume
	container.VolumeMounts = setVolumeMount(container.VolumeMounts)
	podSpec.Containers[0] = container

	// bind /tmp/coverage volume to PVC
	podSpec.Volumes = setVolume(podSpec.Volumes, pvc)
	deployment.Spec.Template.Spec = podSpec

	fmt.Println(deployment.ResourceVersion)
	deployment, err := deploymentClient.Update(ctx, deployment, metav1.UpdateOptions{})
	fmt.Println(deployment.ResourceVersion)

	return err
}

func setEnvVar(env []v1.EnvVar) []v1.EnvVar {
	for _, e := range env {
		if e.Name == "GOCOVERDIR" {
			return env
		}
	}

	return append(env, v1.EnvVar{
		Name:  "GOCOVERDIR",
		Value: mountPath,
	})
}

func setVolumeMount(volumeMounts []v1.VolumeMount) []v1.VolumeMount {
	for _, vm := range volumeMounts {
		if vm.Name == volumeName {
			return volumeMounts
		}
	}

	return append(volumeMounts, v1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
	})
}

func setVolume(volumes []v1.Volume, pvc string) []v1.Volume {
	for _, v := range volumes {
		if v.Name == volumeName {
			return volumes
		}
	}

	return append(
		volumes,
		v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc,
				},
			},
		},
	)
}
