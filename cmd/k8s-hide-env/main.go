package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s-hide-env/k8s"
	"k8s-hide-env/util"
	"log"
	"net/http"

	"gopkg.in/yaml.v2"
)

const secretPrefix = "k8s-hide-env"

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
		return
	}

	var admissionRequest map[string]interface{}
	err = json.Unmarshal(body, &admissionRequest)

	if err != nil {
		log.Printf("Failure to unmarshal admission request: %v", err)
		http.Error(w, "Failure to unmarshal admission request", http.StatusBadRequest)
		return
	}

	request := util.ExtractMap(admissionRequest, "request")
	operation := request["operation"].(string)
	namespace := request["namespace"].(string)
	dryRun := request["dryRun"].(bool)
	resource := extractResource(request)
	apiVersion := admissionRequest["apiVersion"].(string)
	uid := request["uid"].(string)
	var admissionResponseJson string

	if operation == "DELETE" {
		err = deleteSecret(resource, namespace, dryRun, &k8s.KubernetesClientImpl{})
		if err != nil {
			log.Fatalf("Failed to delete k8s-hide-env secret: %v", err)
			http.Error(w, "Failed to delete k8s-hide-env secret", http.StatusBadRequest)
		}
		admissionResponseJson, err = createAdmissionResponse(apiVersion, uid, []map[string]interface{}{})
	} else {
		jsonPatches, err := createJsonPatch(resource, namespace, dryRun, &k8s.KubernetesClientImpl{})
		if err != nil {
			log.Printf("Failure to create json patches: %v", err)
			http.Error(w, "Failure to create json patches", http.StatusInternalServerError)
			return
		}
		admissionResponseJson, err = createAdmissionResponse(apiVersion, uid, jsonPatches)
		if err != nil {
			log.Fatalf("Failed to unmarshal admission request: %v", err)
			http.Error(w, "Failed to unmarshal admission request", http.StatusBadRequest)
		}
	}
	_, err = fmt.Fprintf(w, admissionResponseJson)
	if err != nil {
		log.Printf("Failed to write response body: %v", err)
		http.Error(w, "Can't writing response body", http.StatusInternalServerError)
	}
}

func extractResource(request map[string]interface{}) map[string]interface{} {
	object := util.ExtractMap(request, "object")
	if object == nil {
		object = request["oldObject"].(map[string]interface{})
	}
	return object
}

func createJsonPatch(resource map[string]interface{}, namespace string, dryRun bool, client k8s.KubernetesClient) ([]map[string]interface{}, error) {

	var jsonPatches []map[string]interface{}
	metadata := util.ExtractMap(resource, "metadata")
	name := metadata["name"]
	outerSpec := util.ExtractMap(resource, "spec")
	spec := util.ExtractMap(util.ExtractMap(outerSpec, "template"), "spec")
	volumes := util.ExtractMapList(spec, "volumes")

	// iterate over containers
	containers := util.ExtractMapList(spec, "containers")
	for i := 0; i < len(containers); i++ {
		container := containers[i]
		containerName := container["name"].(string)
		secretName := fmt.Sprintf("%s-%s-%s", secretPrefix, name, containerName)
		envs := util.ExtractMapList(container, "env")
		if envs == nil {
			continue
		}
		var secretData string
		for _, mapEntry := range envs {
			secretData = secretData + "export " + mapEntry["name"].(string) + "=" + mapEntry["value"].(string) + "\n"
		}
		commands := util.ExtractStringList(container, "command")
		args := util.ExtractStringList(container, "args")
		op := "add"
		if len(commands) > 0 {
			op = "replace"
		}
		util.CreatePatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/command", i), []string{"sh", "-c"})
		if len(args) > 0 {
			op = "replace"
		}
		processCommand, err := util.ReconcileShellCommand(commands, args)
		if err != nil {
			return nil, fmt.Errorf("something went wrong with entrypoint command reconciliation: %w", err)
		}
		jsonPatches = append(jsonPatches, util.CreatePatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/command", i), []string{"sh", "-c"}))
		jsonPatches = append(jsonPatches, util.CreatePatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/args", i), []string{processCommand}))
		// if there are no volumes so far
		if volumes == nil {
			jsonPatches = append(jsonPatches, util.CreatePatch("add", "/spec/template/spec/volumes", []map[string]interface{}{{
				"name": secretName,
				"secret": map[string]string{
					"secretName": secretName,
				},
			}}))
		} else {
			volumeIndex := util.GetArrayIndex(volumes, secretName)
			if volumeIndex == -1 {
				jsonPatches = append(jsonPatches, util.CreatePatch("add", "/spec/template/spec/volumes/-", map[string]interface{}{
					"name": secretName,
					"secret": map[string]string{
						"secretName": secretName,
					},
				}))
			}
		}
		volumeMounts := util.ExtractMapList(container, "volumeMounts")
		if volumeMounts == nil {
			jsonPatches = append(jsonPatches, util.CreatePatch(op, fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts", i), []map[string]interface{}{{
				"name":      secretName,
				"mountPath": "/k8s-hide-env",
			}}))
		} else {
			volumeMountIndex := util.GetArrayIndex(volumeMounts, secretName)
			if volumeMountIndex == -1 {
				jsonPatches = append(jsonPatches, util.CreatePatch("add", fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts/-", i), map[string]interface{}{
					"name":      secretName,
					"mountPath": "/k8s-hide-env",
				}))
			}
		}
		jsonPatches = append(jsonPatches, util.CreateRemovePatch(fmt.Sprintf("/spec/template/spec/containers/%d/env", i)))

		// if secret does not exist yet
		if !dryRun {
			secret, err := client.GetSecret(secretName, namespace)
			if err != nil {
				return nil, fmt.Errorf("failure to retrieve k8s-hide-env secret: %w", err)
			}
			if secret == nil {
				err = client.CreateSecret(secretName, namespace, map[string][]byte{"container.env": []byte(secretData)})
				if err != nil {
					return nil, fmt.Errorf("failed to create secret: %w", err)
				}
			} else {
				err = client.ApplySecret(secretName, namespace, map[string][]byte{"container.env": []byte(secretData)})
				if err != nil {
					return nil, fmt.Errorf("failed to apply secret: %w", err)
				}
			}
		}
	}
	return jsonPatches, nil
}

func deleteSecret(resource map[string]interface{}, namespace string, dryRun bool, client k8s.KubernetesClient) error {

	metadata := util.ExtractMap(resource, "metadata")
	name := metadata["name"]
	outerSpec := util.ExtractMap(resource, "spec")
	spec := util.ExtractMap(util.ExtractMap(outerSpec, "template"), "spec")

	// iterate over containers
	containers := util.ExtractMapList(spec, "containers")
	for i := 0; i < len(containers); i++ {
		container := containers[i]
		containerName := container["name"].(string)
		secretName := fmt.Sprintf("%s-%s-%s", secretPrefix, name, containerName)

		// if secret does not exist yet
		log.Printf("Trying to delete secret: %v, in namespace: %v", secretName, namespace)
		secret, err := client.GetSecret(secretName, namespace)
		if err != nil {
			return fmt.Errorf("failure to retrieve k8s-hide-env secret: %w", err)
		}
		if secret != nil && !dryRun {
			log.Printf("Deleting secret: %v, in namespace: %w", secretName, namespace)
			err = client.DeleteSecret(secretName, namespace)
			if err != nil {
				return fmt.Errorf("failed to delete secret: %w", err)
			}
		}
	}
	return nil
}

func createAdmissionResponse(apiVersion string, requestId string, jsonPatches []map[string]interface{}) (string, error) {
	// create admission response
	admissionReviewReturn := map[string]interface{}{}
	admissionReviewReturn["apiVersion"] = apiVersion
	admissionReviewReturn["kind"] = "AdmissionReview"
	admissionResponse := map[string]interface{}{}
	admissionReviewReturn["response"] = admissionResponse
	admissionResponse["uid"] = requestId
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
