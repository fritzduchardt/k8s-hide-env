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
	log.Printf("Starting K8s-Hide-Env on port %d", port)
	err := http.ListenAndServeTLS(fmt.Sprintf(":%d", port), "/cert/tls.crt", "/cert/tls.key", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func mutateHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receiving mutate request")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || string(body) == "" {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Can't read request body", http.StatusBadRequest)
	}
	var client K8sClient
	client = K8sClientImpl{}
	admissionResponseJson, err := createAdmissionResponse(string(body), client)
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

func createAdmissionResponse(admissionRequestJson string, client K8sClient) (string, error) {

	secretPrefix := "k8s-hide-env"
	var jsonPatches []map[string]interface{}

	var admissionRequest map[string]interface{}
	err := json.Unmarshal([]byte(admissionRequestJson), &admissionRequest)
	if err != nil {
		return "", fmt.Errorf("failure to unmarshal admission request: %w", err)
	}
	request := extractMap(admissionRequest, "request")
	namespace := request["namespace"].(string)
	releaseName := request["name"].(string)
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
	volumes := extractList(spec, "volumes")

	// iterate over containers
	containers := extractList(spec, "containers")
	for i := 0; i < len(containers); i++ {
		container := containers[i]
		containerName := container["name"].(string)
		secretName := fmt.Sprintf("%s-%s-%s", secretPrefix, releaseName, containerName)
		envs := extractList(container, "env")
		if envs == nil {
			continue
		}
		var secretData string
		for _, mapEntry := range envs {
			secretData = secretData + "export " + mapEntry["name"].(string) + "=" + mapEntry["value"].(string) + "\n"
		}
		commands := extractStringList(container, "command")
		args := extractStringList(container, "args")
		op := "add"
		if len(commands) > 0 {
			op = "replace"
		}
		createPatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/command", i), []string{"sh", "-c"})
		if len(args) > 0 {
			op = "replace"
		}
		processCommand, err := reconcile(commands, args)
		if err != nil {
			return "", fmt.Errorf("something went wrong with entrypoint command reconciliation: %w", err)
		}
		jsonPatches = append(jsonPatches, createPatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/command", i), []string{"sh", "-c"}))
		jsonPatches = append(jsonPatches, createPatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/args", i), []string{processCommand}))
		// if there are no volumes so far
		if volumes == nil {
			jsonPatches = append(jsonPatches, createPatch("add", "/spec/template/spec/volumes", []map[string]interface{}{{
				"name": secretName,
				"secret": map[string]string{
					"secretName": secretName,
				},
			}}))
		} else {
			volumeIndex := getArrayIndex(volumes, secretName)
			if volumeIndex == -1 {
				jsonPatches = append(jsonPatches, createPatch("add", "/spec/template/spec/volumes/-", map[string]interface{}{
					"name": secretName,
					"secret": map[string]string{
						"secretName": secretName,
					},
				}))
			}
		}
		volumeMounts := extractList(container, "volumeMounts")
		if volumeMounts == nil {
			jsonPatches = append(jsonPatches, createPatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts", i), []map[string]interface{}{{
				"name":      secretName,
				"mountPath": "/k8s-hide-env",
			}}))
		} else {
			volumeMountIndex := getArrayIndex(volumeMounts, secretName)
			if volumeMountIndex == -1 {
				jsonPatches = append(jsonPatches, createPatch("add", fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts/-", i), map[string]interface{}{
					"name":      secretName,
					"mountPath": "/k8s-hide-env",
				}))
			}
		}
		jsonPatches = append(jsonPatches, createRemovePatch(fmt.Sprintf("/spec/template/spec/containers/%d/env", i)))

		// if secret does not exist yet
		secret, err := client.GetSecret(secretName, namespace)
		if err != nil {
			return "", fmt.Errorf("failure to retrieve k8shideenv secret: %w", err)
		}
		if secret == nil {
			err = client.CreateSecret(secretName, namespace, map[string][]byte{"container.env": []byte(secretData)})
			if err != nil {
				return "", fmt.Errorf("failed to create secret: %w", err)
			}
		} else {
			err = client.ApplySecret(secretName, namespace, map[string][]byte{"container.env": []byte(secretData)})
			if err != nil {
				return "", fmt.Errorf("failed to apply secret: %w", err)
			}
		}
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
		return "", fmt.Errorf("failed to encode patches to JSON: %w", err)
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
	vals := container[key]
	if vals != nil {
		var retVal []map[string]interface{}
		for _, entry := range vals.([]interface{}) {
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
	extracted := container[key]
	if extracted != nil {
		return extracted.(map[string]interface{})
	}
	return nil
}

func reconcile(commands []string, args []string) (string, error) {

	if len(commands) == 0 && len(args) == 0 {
		return "", errors.New("either command or args have to be defined in K8s manifest")
	}

	allCommands := []string{". /k8s-hide-env/container.env &&"}
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

func getArrayIndex(mapList []map[string]interface{}, name string) int {
	for index, entry := range mapList {
		if entry["name"] == name {
			return index
		}
	}
	return -1
}
