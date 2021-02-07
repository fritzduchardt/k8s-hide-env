package com.fduchardt.k8shideenv.service;

import com.google.gson.*;
import lombok.extern.slf4j.*;
import org.springframework.beans.factory.annotation.*;
import org.springframework.stereotype.*;
import org.yaml.snakeyaml.*;

import java.nio.charset.*;
import java.util.*;
import java.util.stream.*;

@Component
@Slf4j
public class K8sHideEnvService {

    private final Yaml yaml = new Yaml();
    private final Gson gson = new Gson();
    private final CommandAndArgsUtils commandAndArgsUtils;

    public K8sHideEnvService(CommandAndArgsUtils commandAndArgsUtils) {
        this.commandAndArgsUtils = commandAndArgsUtils;
    }

    public String createAdmissionResponse(String admissionRequestJson) {

        List<Map<String, Object>> jsonPatches = new ArrayList<>();

        // extract object or old object (create & update)
        Map<String, Object> admissionRequest = yaml.load(admissionRequestJson);
        Map<String, Object> request = extractMap(admissionRequest, "request");
        Map<String, Object> object = extractMap(request, "object");
        if (object.isEmpty()) {
            object = extractMap(request, "oldObject");
        }
        String kind = (String) object.get("kind");
        if (
                !kind.equals("Deployment") &&
                        !kind.equals("DaemonSet") &&
                        !kind.equals("StatefulSet")
        ) {
            throw new RuntimeException("Currently, only supports Deployments, Daemonsets and StatefulSets.");
        }
        Map<String, Object> outerSpec = extractMap(object, "spec");
        Map<String, Object> spec = extractMap(extractMap(outerSpec, "template"), "spec");

        // iterate over containers
        List<Map<String, Object>> containers = extractList(spec, "containers");
        for (int i = 0; i < containers.size(); i++) {
            Map<String, Object> container = containers.get(i);

            // no envs, no action required
            List<Map<String, Object>> envs = extractList(container, "env");
            if (envs.isEmpty()) {
                continue;
            }

            // add init-containers to save envs in file
            List<String> envCommands = new ArrayList<>();
            envCommands.add("env");
            for (Map<String, Object> env : envs) {
                StringBuilder envCmd = new StringBuilder();
                envCmd.append(env.get("name"));
                envCmd.append("=");
                String value = (String) env.get("value");
                if (value == null) {
                    throw new RuntimeException("Currently, only support environment values written straight to the deployment definition.");
                }
                envCmd.append(value);
                envCommands.add(envCmd.toString());
            }
            envCommands.add("sh");
            envCommands.add("-c");

            List<String> commands = extractList(container, "command");
            List<String> args = extractList(container, "args");
            jsonPatches.add(createPatch(commands.isEmpty() ? "add" : "replace", "/spec/template/spec/containers/" + i + "/command", envCommands));
            jsonPatches.add(createPatch(args.isEmpty() ? "add" : "replace", "/spec/template/spec/containers/" + i + "/args", List.of(commandAndArgsUtils.reconcile(commands, args))));

            // delete container envs
            jsonPatches.add(createRemovePatch("/spec/template/spec/containers/" + i + "/env"));
        }

        // create admission response
        Map<String, Object> admissionReviewReturn = new HashMap<>();
        admissionReviewReturn.put("apiVersion", admissionRequest.get("apiVersion"));
        admissionReviewReturn.put("kind", "AdmissionReview");
        Map<String, Object> admissionResponse = new HashMap<>();
        admissionReviewReturn.put("response", admissionResponse);
        admissionResponse.put("uid", request.get("uid"));
        admissionResponse.put("allowed", true);
        admissionResponse.put("patchType", "JSONPatch");
        String json = gson.toJson(jsonPatches);
        log.info(json);
        admissionResponse.put("patch", Base64.getEncoder().encodeToString(json.getBytes(StandardCharsets.UTF_8)));

        return yaml.dump(admissionReviewReturn);
    }

    private Map<String, Object> createRemovePatch(String path) {
        return Map.of("op", "remove", "path", path);
    }

    private Map<String, Object> createPatch(String op, String path, Object value) {
        return Map.of("op", op, "path", path, "value", value);
    }

    @SuppressWarnings({"unchecked"})
    private <T> List<T> extractList(Map<String, Object> container, String key) {
        List<T> strings = (List<T>) container.get(key);
        return strings == null ? List.of() : strings;
    }

    @SuppressWarnings({"unchecked"})
    private <K, O> Map<K, O> extractMap(Map<String, Object> container, String key) {
        Map<K, O> mapList = (Map<K, O>) container.get(key);
        return mapList == null ? Map.of() : mapList;
    }
}
