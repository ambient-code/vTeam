package services

import (
	"context"

	"ambient-code-operator/internal/config"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// EnsureProjectWorkspacePVC creates a per-namespace PVC for runner workspace if missing
func EnsureProjectWorkspacePVC(namespace string) error {
	// Check if PVC exists
	if _, err := config.K8sClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), "ambient-workspace", v1.GetOptions{}); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ambient-workspace",
			Namespace: namespace,
			Labels:    map[string]string{"app": "ambient-workspace"},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("5Gi"),
				},
			},
		},
	}
	if _, err := config.K8sClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

// EnsureContentService deploys a per-namespace content service that mounts the project PVC RW
func EnsureContentService(namespace string) error {
	appConfig := config.LoadConfig()

	// Check Service
	if _, err := config.K8sClient.CoreV1().Services(namespace).Get(context.TODO(), "ambient-content", v1.GetOptions{}); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	// Deployment
	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ambient-content",
			Namespace: namespace,
			Labels:    map[string]string{"app": "ambient-content"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &v1.LabelSelector{MatchLabels: map[string]string{"app": "ambient-content"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{Labels: map[string]string{"app": "ambient-content"}},
				Spec: corev1.PodSpec{
					// Keep content service singleton for RWO PVC; rely on runner job podAffinity (set below) to co-locate with content if needed
					Containers: []corev1.Container{
						{
							Name:  "content",
							Image: appConfig.ContentServiceImage,
							Env: []corev1.EnvVar{
								{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
								{Name: "CONTENT_SERVICE_MODE", Value: "true"},
								{Name: "STATE_BASE_DIR", Value: "/data"},
							},
							Ports:        []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}},
							VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/data"}},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "workspace", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "ambient-workspace"}}},
					},
				},
			},
		},
	}
	if _, err := config.K8sClient.AppsV1().Deployments(namespace).Create(context.TODO(), deploy, v1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// Service
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ambient-content",
			Namespace: namespace,
			Labels:    map[string]string{"app": "ambient-content"},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "ambient-content"},
			Ports:    []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstrFromString("http")}},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	if _, err := config.K8sClient.CoreV1().Services(namespace).Create(context.TODO(), svc, v1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

var (
	intstrFromString = func(s string) intstr.IntOrString { return intstr.Parse(s) }
)