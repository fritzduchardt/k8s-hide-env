package main

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sClient interface {
	CreateSecret(secretName string, namespace string, data map[string][]byte) error
	ApplySecret(secretName string, namespace string, data map[string][]byte) error
	GetSecret(secretName string, namespace string) (*apiv1.Secret, error)
}

type K8sClientImpl struct {
}

func (k8s K8sClientImpl) createClient() (*kubernetes.Clientset, error) {
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

func (k8s K8sClientImpl) CreateSecret(secretName string, namespace string, data map[string][]byte) error {

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

func (k8s K8sClientImpl) ApplySecret(secretName string, namespace string, data map[string][]byte) error {

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

func (k8s K8sClientImpl) GetSecret(secretName string, namespace string) (*apiv1.Secret, error) {

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
