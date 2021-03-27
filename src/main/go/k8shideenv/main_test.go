package main

import (
	"io/ioutil"
	"testing"
)

func TestCreateAdmissionResponse(t *testing.T) {
	json, err := ioutil.ReadFile("../../../test/resources/admission-request.yaml")
	if err != nil {
		t.Errorf("Can't open file: %v", err)
	}
	response, err := createAdmissionResponse(string(json))
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if response == "" {
		t.Errorf("Invalid response %v", response)
		return
	}
}
