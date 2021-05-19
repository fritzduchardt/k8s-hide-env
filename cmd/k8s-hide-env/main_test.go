package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	apiv1 "k8s.io/api/core/v1"
	"testing"
)

type KubernetesClientMock struct {
	CreateCounter int
	ApplyCounter  int
	GetCounter    int
}

func (k8s *KubernetesClientMock) CreateSecret(secretName string, namespace string, data map[string][]byte) error {
	k8s.CreateCounter++
	return nil
}
func (k8s *KubernetesClientMock) ApplySecret(secretName string, namespace string, data map[string][]byte) error {
	k8s.ApplyCounter++
	return nil
}

func (k8s *KubernetesClientMock) GetSecret(secretName string, namespace string) (*apiv1.Secret, error) {
	k8s.GetCounter++
	return &apiv1.Secret{}, nil
}

func TestCreateJsonPatch(t *testing.T) {
	admissionRequestJson, err := readFixture("../../test/admission-request.yaml")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	var admissionRequest map[string]interface{}
	err = json.Unmarshal(admissionRequestJson, &admissionRequest)

	client := &KubernetesClientMock{}
	patch, err := createJsonPatch(admissionRequest, client)

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

	if client.GetCounter != 1 {
		t.Errorf("K8s client getSecret was not invoked as expected: %v, expected number of invocations: %v", client.GetCounter, 1)
	}
	if client.ApplyCounter != 1 {
		t.Errorf("K8s client applySecret was not invoked as expected: %v, expected number of invocations: %v", client.ApplyCounter, 1)
	}
	if client.CreateCounter != 0 {
		t.Errorf("K8s client createSecret was not invoked as expected: %v, expected number of invocations: %v", client.CreateCounter, 0)
	}
}

func TestCreateJsonPatchDryRun(t *testing.T) {
	admissionRequestJson, err := readFixture("../../test/admission-request-dryrun.yaml")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	var admissionRequest map[string]interface{}
	err = json.Unmarshal(admissionRequestJson, &admissionRequest)

	client := &KubernetesClientMock{}
	patch, err := createJsonPatch(admissionRequest, client)

	if err != nil {
		t.Error(err)
		return
	}
	if len(patch) != 5 {
		t.Errorf("Invalid patch length: %v, expected %v", len(patch), 5)
	}

	if client.GetCounter != 0 {
		t.Errorf("K8s client getSecret was not invoked as expected: %v, expected number of invocations: %v", client.GetCounter, 0)
	}
	if client.ApplyCounter != 0 {
		t.Errorf("K8s client applySecret was not invoked as expected: %v, expected number of invocations: %v", client.ApplyCounter, 0)
	}
	if client.CreateCounter != 0 {
		t.Errorf("K8s client createSecret was not invoked as expected: %v, expected number of invocations: %v", client.CreateCounter, 0)
	}
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
