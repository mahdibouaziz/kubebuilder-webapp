package controllers

import (
	"fmt"
	"net"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	webappv1 "github.com/mahdibouaziz/kubebuilder-webapp/api/v1"
)

func (r *GuestbookReconciler) desiredDeployment(book webappv1.Guestbook, redis webappv1.Redis) (appsv1.Deployment, error) {

	depl := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      book.Name,
			Namespace: book.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: book.Spec.Frontend.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"guestbook": book.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"guestbook": book.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "frontend",
							Image: "gcr.io/google-samples/gb-frontend:v4",
							Env: []corev1.EnvVar{
								{Name: "GET_HOSTS_FROM", Value: "env"},
								{Name: "REDIS_MASTER_SERVICE_HOST", Value: redis.Status.LeaderService},
								{Name: "REDIS_SLAVE_SERVICE_HOST", Value: redis.Status.FollowerService},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 80, Name: "http", Protocol: "TCP"},
							},
							Resources: *book.Spec.Frontend.Resources.DeepCopy(),
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(&book, &depl, r.Scheme); err != nil {
		return depl, err
	}

	return depl, nil
}

func (r *GuestbookReconciler) desiredService(book webappv1.Guestbook) (corev1.Service, error) {

	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      book.Name,
			Namespace: book.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					Protocol:   "TCP",
					TargetPort: intstr.FromString("http"),
				},
			},
			Selector: map[string]string{"guestbook": book.Name},
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	}

	// always set the controller reference so that we know which object owns this.
	if err := ctrl.SetControllerReference(&book, &svc, r.Scheme); err != nil {
		return svc, err
	}

	return svc, nil
}

func urlForService(svc corev1.Service, port int32) string {
	// notice that we unset this if it's not present -- we always want the
	// state to reflect what we observe.
	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return ""
	}

	host := svc.Status.LoadBalancer.Ingress[0].Hostname
	if host == "" {
		host = svc.Status.LoadBalancer.Ingress[0].IP
	}

	return fmt.Sprintf("http://%s", net.JoinHostPort(host, fmt.Sprintf("%v", port)))
}
