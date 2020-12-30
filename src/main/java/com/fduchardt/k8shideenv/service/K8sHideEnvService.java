package com.fduchardt.k8shideenv.service;

import com.google.gson.*;
import lombok.extern.slf4j.*;
import org.springframework.stereotype.*;
import org.yaml.snakeyaml.*;

import java.nio.charset.*;
import java.util.*;
import java.util.stream.*;

@Component
@Slf4j
public class K8sHideEnvService {

    Yaml yaml = new Yaml();
    Gson gson = new Gson();

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

        // add tmpfs volume to temporarily store credentials
        List<Map<String, Object>> volumes = new ArrayList<>();
        Map<String, Object> volume = new HashMap<>();
        volumes.add(volume);
        volume.put("name", "k8s-hide-env");
        volume.put("emptyDir", Map.of("medium", "Memory"));
        if (spec.containsKey("volumes")) {
            jsonPatches.add(createPatch("add", "/spec/template/spec/volumes/-", volume));
        } else {
            jsonPatches.add(createPatch("add", "/spec/template/spec/volumes", volumes));
        }

        // iterate over containers
        List<Map<String, Object>> containers = extractList(spec, "containers");
        for (int i = 0; i < containers.size(); i++) {
            Map<String, Object> container = containers.get(i);

            // no envs, no action required
            List<Map<String, Object>> envs = extractList(container, "env");
            if (envs.isEmpty()) {
                break;
            }

            // add init-containers to save envs in file
            StringBuilder envCmd = new StringBuilder();
            envCmd.append("echo '");
            for (Map<String, Object> env : envs) {
                envCmd.append("export " + env.get("name"));
                envCmd.append("=");
                String value = (String) env.get("value");
                if (value == null) {
                    throw new RuntimeException("Currently, only support environment values written straight to the deployment definition.");
                }
                envCmd.append(value);
                envCmd.append("\\n");
            }
            envCmd.append("' > /envs/hide-env-" + i + ".sh");
            List<String> sh = List.of("sh", "-c", envCmd.toString());
            List<Map<String, Object>> initContainers = new ArrayList<>();
            Map<String, Object> initContainer = new HashMap<>();
            initContainers.add(initContainer);
            initContainer.put("command", sh);
            initContainer.put("name", "k8s-hide-env-" + i);
            initContainer.put("image", "ubuntu");
            List<Map<String, Object>> volumeMounts = new ArrayList<>();
            initContainer.put("volumeMounts", volumeMounts);
            Map<String, Object> volumeMount = new HashMap<>();
            initContainer.put("volumeMounts", volumeMounts);
            volumeMount.put("mountPath", "/envs");
            volumeMount.put("name", "k8s-hide-env");
            volumeMounts.add(volumeMount);
            if (spec.containsKey("initContainers") || i > 0) {
                jsonPatches.add(createPatch("add", "/spec/template/spec/initContainers/-", initContainer));
            } else {
                jsonPatches.add(createPatch("add", "/spec/template/spec/initContainers", initContainers));
            }

            // mount tmpfs volume into container
            List<Map<String, Object>> containerMounts = new ArrayList<>();
            Map<String, Object> containerMount = new HashMap<>();
            containerMounts.add(containerMount);
            containerMount.put("mountPath", "/envs");
            containerMount.put("name", "k8s-hide-env");
            if (container.containsKey("volumeMounts")) {
                jsonPatches.add(createPatch("add", "/spec/template/spec/containers/" + i + "/volumeMounts/-", containerMount));
            } else {
                jsonPatches.add(createPatch("add", "/spec/template/spec/containers/" + i + "/volumeMounts", containerMounts));
            }

            // overwrite container command with "sh"
            jsonPatches.add(createPatch("replace", "/spec/template/spec/containers/" + i + "/command", List.of("sh", "-c")));

            // add sourcing of env files to container args
            List<String> commands = filterShell(extractList(container, "command"));
            List<String> args = filterShell(extractList(container, "args"));
            List<String> newArgs = Stream.concat(commands.stream(), args.stream()).collect(Collectors.toList());
            if (newArgs.isEmpty()) {
                throw new RuntimeException("Currently, work of image defaults. Need to specify command, args or both.");
            }
            jsonPatches.add(createPatch("replace", "/spec/template/spec/containers/" + i + "/args", List.of(". /envs/hide-env-" + i + ".sh && rm /envs/hide-env-" + i + ".sh && "+  String.join(" ", newArgs))));

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

    private List<String> filterShell(List<String> oldCommands) {
        List<String> filteredOldCommands;
        if (oldCommands.size() > 1 && oldCommands.get(0).equals("sh") && oldCommands.get(1).equals("-c")) {
            filteredOldCommands = oldCommands.stream().skip(2).collect(Collectors.toList());
        } else {
            filteredOldCommands = oldCommands.stream().collect(Collectors.toList());
        }
        filteredOldCommands.forEach(command -> {
            if (command.equals("sh")) {
                throw new RuntimeException("Currently can't handle nested shell commands.");
            }
        });
        return filteredOldCommands;
    }

    private Map<String, Object> createRemovePatch(String path) {
        return Map.of("op", "remove", "path", path);
    }

    private Map<String, Object> createPatch(String op, String path, Object value) {
        return Map.of("op", op, "path", path, "value", value);
    }

    private <T> List<T> extractList(Map<String, Object> container, String key) {
        List<T> strings = (List<T>) container.get(key);
        return strings == null ? List.of() : strings;
    }

    private <K, O> Map<K, O> extractMap(Map<String, Object> container, String key) {
        Map<K, O> mapList = (Map<K, O>) container.get(key);
        return mapList == null ? Map.of() : mapList;
    }
}
