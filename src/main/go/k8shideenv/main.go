package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"gopkg.in/yaml.v2"
)

func main() {
	http.HandleFunc("/mutate", mutateHandler)
	port := 8443
	log.Printf("Starting K8s Hide-Env on port %d", port)
	err := http.ListenAndServeTLS(fmt.Sprintf(":%d", port), "/cert/tls.crt", "/cert/tls.key", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func mutateHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || string(body) == "" {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Can't read request body", http.StatusBadRequest)
	}
	admissionResponseJson, err := createAdmissionResponse(string(body))
	if err != nil {
		log.Fatalf("Failed to unmarshal admission request: %v", err)
		http.Error(w, "Failed to unmarshal admission request", http.StatusBadRequest)
	}
	_, err = fmt.Fprintf(w, admissionResponseJson)
	if err != nil {
		log.Printf("Failed to write response body: %v", err)
		http.Error(w, "Can't writing response body", http.StatusInternalServerError)
	}
}

func createAdmissionResponse(admissionRequestJson string) (string, error) {
	var jsonPatches []map[string]interface{}

	var admissionRequest map[string]interface{}
	err := json.Unmarshal([]byte(admissionRequestJson), &admissionRequest)
	if err != nil {
		return "", fmt.Errorf("failure to unmarshal admission request: %w", err)
	}
	request := extractMap(admissionRequest, "request")
	object := extractMap(request, "object")
	if object == nil {
		object = request["oldObject"].(map[string]interface{})
	}
	kind := object["kind"].(string)
	if kind != "Deployment" && kind != "DaemonSet" && kind != "StatefulSet" {
		return "", errors.New("unsupported resource type")
	}
	outerSpec := extractMap(object, "spec")
	spec := extractMap(extractMap(outerSpec, "template"), "spec")

	// iterate over containers
	containers := extractList(spec, "containers")
	for i := 0; i < len(containers); i++ {
		container := containers[i]
		envs := extractList(container, "env")
		//	List<Map<String, Object>> envs = extractList(container, "env");
		if envs == nil {
			continue
		}
		// add init-containers to save envs in file
		var envCommands []string
		envCommands = append(envCommands, "env")
		for _, mapEntry := range envs {
			var envCmd string
			name := mapEntry["name"].(string)
			envCmd += name
			envCmd += "="
			envCmd += mapEntry["value"].(string)
			envCommands = append(envCommands, envCmd)
		}
		envCommands = append(envCommands, "sh", "-c")

		commands := extractStringList(container, "command")
		args := extractStringList(container, "args")
		op := "add"
		if len(commands) > 0 {
			op = "replace"
		}
		createPatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/command", i), envCommands)
		if len(args) > 0 {
			op = "replace"
		}
		processCommand, err := reconcile(commands, args)
		if err != nil {
			return "something went wrong with entrypoint command reconciliation", err
		}

		jsonPatches = append(jsonPatches, createPatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/command", i), envCommands))
		jsonPatches = append(jsonPatches, createPatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/args", i), []string{processCommand}))
		jsonPatches = append(jsonPatches, createRemovePatch(fmt.Sprintf("/spec/template/spec/containers/%d/env", i)))
	}

	// create admission response
	admissionReviewReturn := map[string]interface{}{}
	admissionReviewReturn["apiVersion"] = admissionRequest["apiVersion"]
	admissionReviewReturn["kind"] = "AdmissionReview"
	admissionResponse := map[string]interface{}{}
	admissionReviewReturn["response"] = admissionResponse
	admissionResponse["uid"] = request["uid"]
	admissionResponse["allowed"] = true
	admissionResponse["patchType"] = "JSONPatch"
	json, err := json.Marshal(jsonPatches)
	if err != nil {
		return "failed to encode patches to JSON", err
	}
	log.Printf(string(json))
	admissionResponse["patch"] = b64.StdEncoding.EncodeToString(json)
	admissionResponseJson, err := yaml.Marshal(&admissionReviewReturn)
	log.Printf(string(admissionResponseJson))
	return string(admissionResponseJson), nil
}

func createRemovePatch(path string) map[string]interface{} {
	return map[string]interface{}{
		"op":   "remove",
		"path": path,
	}
}

func createPatch(op string, path string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"op":    op,
		"path":  path,
		"value": value,
	}
}

func extractList(container map[string]interface{}, key string) []map[string]interface{} {
	vals := container[key].([]interface{})
	var retVal []map[string]interface{}
	if vals != nil {
		for _, entry := range vals {
			retVal = append(retVal, entry.(map[string]interface{}))
		}
		return retVal
	}
	return nil
}

func extractStringList(container map[string]interface{}, key string) []string {
	values := container[key]
	if values == nil {
		return nil
	}
	var retVal []string
	for _, val := range values.([]interface{}) {
		retVal = append(retVal, val.(string))
	}
	return retVal
}

func extractMap(container map[string]interface{}, key string) map[string]interface{} {
	extracted := container[key].(map[string]interface{})
	if extracted != nil {
		return extracted
	}
	return nil
}

func reconcile(commands []string, args []string) (string, error) {

	if len(commands) == 0 && len(args) == 0 {
		return "", errors.New("either command or args have to be defined in K8s manifest")
	}

	var allCommands []string
	allCommands = append(allCommands, removeShellCommand(commands)...)
	allCommands = append(allCommands, removeShellCommand(args)...)
	return strings.Join(allCommands, " "), nil
}

func removeShellCommand(oldCommands []string) []string {

	if len(oldCommands) > 1 && oldCommands[0] == "sh" && oldCommands[1] == "-c" {
		return oldCommands[2:]
	}
	return oldCommands
}
