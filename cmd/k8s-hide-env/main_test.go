package main

import (
	"io/ioutil"
	apiv1 "k8s.io/api/core/v1"
	"testing"
)

type K8sClientMock struct {
}

func (k8s K8sClientMock) CreateSecret(secretName string, namespace string, data map[string][]byte) error {
	return nil
}
func (k8s K8sClientMock) ApplySecret(secretName string, namespace string, data map[string][]byte) error {
	return nil
}

func (k8s K8sClientMock) GetSecret(secretName string, namespace string) (*apiv1.Secret, error) {
	return &apiv1.Secret{}, nil
}

func TestCreateAdmissionResponse(t *testing.T) {
	json, err := ioutil.ReadFile("../../test/admission-request.yaml")
	if err != nil {
		t.Errorf("Can't open file: %v", err)
	}
	response, err := createAdmissionResponse(string(json), K8sClientMock{})
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if response == "" {
		t.Errorf("Invalid response %v", response)
		return
	}
}
