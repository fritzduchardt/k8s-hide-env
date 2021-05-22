package k8s

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubernetesClient interface {
	CreateSecret(secretName string, namespace string, data map[string][]byte) error
	ApplySecret(secretName string, namespace string, data map[string][]byte) error
	GetSecret(secretName string, namespace string) (*apiv1.Secret, error)
	DeleteSecret(secretName string, namespace string) error
}

type KubernetesClientImpl struct {
}

func (k8s *KubernetesClientImpl) createClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()

	if err != nil {
		return nil, fmt.Errorf("Can't retrieve cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Can't create k8s client: %w", err)
	}
	return clientset, nil
}

func (k8s *KubernetesClientImpl) CreateSecret(secretName string, namespace string, data map[string][]byte) error {

	clientset, err := k8s.createClient()
	if err != nil {
		return err
	}

	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: data,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{
		FieldManager: "k8s-hide-env",
	})
	if err != nil {
		panic(err.Error())
	}
	return nil
}

func (k8s *KubernetesClientImpl) ApplySecret(secretName string, namespace string, data map[string][]byte) error {

	clientset, err := k8s.createClient()
	if err != nil {
		return err
	}

	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: data,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{
		FieldManager: "k8s-hide-env",
	})
	if err != nil {
		panic(err.Error())
	}
	return nil
}

func (k8s *KubernetesClientImpl) GetSecret(secretName string, namespace string) (*apiv1.Secret, error) {

	clientset, err := k8s.createClient()
	if err != nil {
		return nil, err
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, nil
	}

	return secret, nil
}

func (k8s *KubernetesClientImpl) DeleteSecret(secretName string, namespace string) error {

	clientset, err := k8s.createClient()
	if err != nil {
		return err
	}

	err = clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), secretName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
