package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s-hide-env/k8s"
	"k8s-hide-env/util"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/mock"
)

type KubernetesClientMock struct {
	mock.Mock
}

func (m *KubernetesClientMock) CreateSecret(secretName string, namespace string, data map[string][]byte) error {
	args := m.Called(secretName, namespace, data)
	return args.Error(0)
}
func (k8s *KubernetesClientMock) ApplySecret(secretName string, namespace string, data map[string][]byte) error {
	return nil
}

func (k8s *KubernetesClientMock) GetSecret(secretName string, namespace string) (*apiv1.Secret, error) {
	return &apiv1.Secret{}, nil
}

func (k8s *KubernetesClientMock) DeleteSecret(secretName string, namespace string) error {
	return nil
}

func TestCreateJsonPatch(t *testing.T) {
	admissionRequestJson, err := readFixture("../../test/admission-request.yaml")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	var admissionRequest map[string]interface{}
	err = json.Unmarshal(admissionRequestJson, &admissionRequest)
	request := util.ExtractMap(admissionRequest, "request")
	resource := extractResource(request)
	namespace := request["namespace"].(string)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := k8s.NewMockKubernetesClient(ctrl)

	client.EXPECT().GetSecret("k8s-hide-env-k8sshowcase-k8s-showcase-application", "default").Times(1)
	client.EXPECT().CreateSecret(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	patch, err := createJsonPatch(resource, namespace, false, client)

	if err != nil {
		t.Error(err)
		return
	}
	if len(patch) != 5 {
		t.Errorf("Invalid patch length: %v, expected %v", len(patch), 5)
	}
	assertPatch(t, patch[0], "replace", "/spec/template/spec/containers/0/command")
	assertPatch(t, patch[1], "replace", "/spec/template/spec/containers/0/args")
	assertPatch(t, patch[2], "add", "/spec/template/spec/volumes")
	assertPatch(t, patch[3], "replace", "/spec/template/spec/containers/0/volumeMounts")
	assertPatch(t, patch[4], "remove", "/spec/template/spec/containers/0/env")

}

func TestCreateJsonPatchDryRun(t *testing.T) {
	admissionRequestJson, err := readFixture("../../test/admission-request.yaml")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	var admissionRequest map[string]interface{}
	err = json.Unmarshal(admissionRequestJson, &admissionRequest)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := k8s.NewMockKubernetesClient(ctrl)

	request := util.ExtractMap(admissionRequest, "request")
	resource := extractResource(request)
	namespace := request["namespace"].(string)
	patch, err := createJsonPatch(resource, namespace, true, client)

	if err != nil {
		t.Error(err)
		return
	}
	if len(patch) != 5 {
		t.Errorf("Invalid patch length: %v, expected %v", len(patch), 5)
	}
}

func TestDeleteSecret(t *testing.T) {
	admissionRequestJson, err := readFixture("../../test/admission-request.yaml")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	var admissionRequest map[string]interface{}
	err = json.Unmarshal(admissionRequestJson, &admissionRequest)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := k8s.NewMockKubernetesClient(ctrl)
	client.EXPECT().GetSecret("k8s-hide-env-k8sshowcase-k8s-showcase-application", "default").Return(createSecret(), nil)
	client.EXPECT().DeleteSecret("k8s-hide-env-k8sshowcase-k8s-showcase-application", "default").Times(1)

	request := util.ExtractMap(admissionRequest, "request")
	resource := extractResource(request)
	namespace := request["namespace"].(string)
	err = deleteSecret(resource, namespace, false, client)

	if err != nil {
		t.Error(err)
		return
	}
}

func TestDeleteSecretDryRun(t *testing.T) {
	admissionRequestJson, err := readFixture("../../test/admission-request.yaml")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	var admissionRequest map[string]interface{}
	err = json.Unmarshal(admissionRequestJson, &admissionRequest)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := k8s.NewMockKubernetesClient(ctrl)
	client.EXPECT().GetSecret("k8s-hide-env-k8sshowcase-k8s-showcase-application", "default").Return(createSecret(), nil)

	request := util.ExtractMap(admissionRequest, "request")
	resource := extractResource(request)
	namespace := request["namespace"].(string)
	err = deleteSecret(resource, namespace, true, client)

	if err != nil {
		t.Error(err)
		return
	}
}

func createSecret() *apiv1.Secret {
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-name",
			Namespace: "default",
		},
		Data: nil,
	}
	return secret
}

func assertPatch(t *testing.T, patch map[string]interface{}, op string, path string) {
	if patch["op"] != op {
		t.Errorf("Invalid op: %v, expected: %v", patch["op"], op)
	}
	if patch["path"] != path {
		t.Errorf("Invalid path: %v, expected: %v", patch["path"], path)
	}
}

func TestCreateAdmissionResponse(t *testing.T) {
	jsonPatches := []map[string]interface{}{
		{
			"key": "value",
		},
	}
	apiVersion := "v1"
	responseId := "123"
	response, err := createAdmissionResponse(apiVersion, responseId, jsonPatches)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if response == "" || &response == nil {
		t.Errorf("Invalid response %v", response)
		return
	}

	var admissionResponse map[string]interface{}
	err = yaml.Unmarshal([]byte(response), &admissionResponse)
	if err != nil {
		t.Errorf("Error: %v. Failed to unmarshal admission response: %v", err, response)
		return
	}
	if admissionResponse["apiVersion"] != apiVersion {
		t.Errorf("Invalid apiVersion %v, expected: %v", admissionResponse["apiVersion"], apiVersion)
	}
	if admissionResponse["kind"] != "AdmissionReview" {
		t.Errorf("Invalid kind %v, expected: AdmissionReview", admissionResponse["kind"])
	}
	nestedResponse := admissionResponse["response"].(map[interface{}]interface{})

	if nestedResponse["uid"] != responseId {
		t.Errorf("Invalid uid %v, expected %v", nestedResponse["uid"], responseId)
	}
	if nestedResponse["allowed"] != true {
		t.Errorf("Invalid allowed %v, expected: true", nestedResponse["allowed"])
	}
	if nestedResponse["patchType"] != "JSONPatch" {
		t.Errorf("Invalid patchType %v, expected: JSONPatch", nestedResponse["patchType"])
	}

	encodedPatches, err := json.Marshal(jsonPatches)
	encodedPatchStr := b64.StdEncoding.EncodeToString(encodedPatches)
	if nestedResponse["patch"] != encodedPatchStr {
		t.Errorf("Invalid json patches %v, expected %v", nestedResponse["patch"], encodedPatchStr)
	}
}

func readFixture(path string) ([]byte, error) {
	json, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}
	return json, err
}
