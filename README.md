# K8s Hide Env

K8s Hide Env removes environment variables from your container environment and makes them  **only visible to the container main process and its child processes**.

By default, K8s exposes environment variables in the container environment. This carries **substantial security risks**, since it allows container hijackers to gain access to infrastructure credentials.

## How it works

K8s Hide Env installs a [Mutating Web Hook](https://kubernetes.io/blog/2019/03/21/a-guide-to-kubernetes-admission-controllers/) in your K8s cluster, which in a nut-shell **moves your container environment variables to an in-memory file that is sourced by the shell that starts the main process and then deleted**. In detail, the following changes are made to the K8s manifests of your *Deployments*, *Daemonsets* or *StatefulSets*:

1. Add an in-memory empty directory to the Pod template.
2. Add an init container for each Pod container that writes the container env variables to a file. This file is written to the in-memory directory.
3. Mount the in-memory directory into each container.
4. Amend the container startup command to first source the env variables from that file, before starting the container main process. The file is deleted immediately after sourcing.
5. Remove all environment variables from the K8s manifest.

## Limitations

- Only works on *Deployments*, *StatefulSets* and *Daemonsets*.
- All environment variables have to be written straight into the K8s manifests. Reading from *Secrets* or *ConfigMaps* is currently not supported.
- `ENTRYPOINT` and / or `CMD` configuration of the application container image has to get overwritten in K8s manifest with the `command` and / or `args` element.
- Environment values will still be visible from the worker node proc file system.
- Environment values will still be visible in the K8s manifest.

## Installation

### Prerequisits

Internal K8s traffic between infrastructure components is mandated to use TLS - therefore your need:

- A server private key PEM file.
- A certificate signing request CSR file.

### Obtain a signed certificate

#### Step 1 - Create K8s CertificateSigningRequest:
```shell
cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: webhook.default
spec:
  request: $(cat server.csr | base64 | tr -d '\n')
  signerName: kubernetes.io/kubelet-serving
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF
```
#### Step 2 - Approve CertificateSigningRequest:
```shell
kubectl certificate approve webhook.default
```
#### Step 3 - Download the approved certificate:
```shell
kubectl get csr webhook.default -o jsonpath='{.status.certificate}' \
| base64 --decode > server.crt
```
#### Step 4 - Install the certificate as a TLS secret in your K8s cluster:
```shell
kubectl create secret tls webhook-tls --cert=server.crt --key=server-key.pem
```

### Install K8s Hide Env
```shell
kubectl apply -f src/main/k8s/service.yaml
kubectl apply -f src/main/k8s/k8s-hide-env-deployment.yaml
```

### Create the Webhook

Please note that here you have to provide the K8s cluster root certificate to validate TLS connections. E.g. with Minikube, this can be done as follows:

```shell
cat <<EOF | kubectl replace --force -f -
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: webhook
  labels:
    app: webhook
webhooks:
  - name: webhook.default.svc.cluster.local
    sideEffects: None
    admissionReviewVersions: ["v1", "v1beta1"]
    matchPolicy: Equivalent
    objectSelector:
      matchLabels:
        mode: secure
    clientConfig:
      caBundle: $(cat ~/.minikube/ca.crt | base64 -w0)
      service:
        name: webhook
        namespace: default
        path: "/mutate"
        port: 8443
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["*"]
        apiVersions: ["*"]
        resources: ["deployments", "daemonsets", "statefulsets"]
        scope: "*"
EOF
```

### Try it out

Now, install a *Deployment* to your cluster, e.g. the [K8s Showcase App](https://github.com/fritzduchardt/k8s-showcase-application):
```shell
kubectl apply -f src/test/resources/deploy.yaml
```
Note, that when looking at the container environment, the environment variable `MESSAGE` is not visible:
```shell
kubectl exec k8sshowcase-76cd657458-mg6xp env | grep MESSAGE
> 
```
However, the app has an endpoint to expose environment variables and here `MESSAGE` can be seen:
```
# open connection to one of your Pods
kubectl port-forward k8sshowcase-76cd657458-fm8k5 8080
# curl env endpoint
curl localhost:8080/env/MESSAGE
> Test
```

## Feedback

Any ideas and / or feedback regarding this project is very welcome. Please write to [fritz@duchardt.net](mailto:fritz@duchardt.net).