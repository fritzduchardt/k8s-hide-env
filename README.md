# K8s Hide Env

## Description

K8s commonly exposes environment variables in the container environment. This carries a substantial security risk, since it allows container hijackers to gain access to infrastructure credentials.

K8s Hide Env addresses this problem by making environment variables only visible to the container main process.

## How it works

K8s Hide Env adds a Mutating Web Hook to your K8s cluster that modifies your Deployments as follows:

1. Add an in-memory empty directory to your Pod(s).
2. Add an init container for each Pod container that writes the container env variables to a file. This file is placed in the in-memory directory.
3. Mount the in-memory directory into each container.
4. Amend the container startup command to first source the env variables from that file, before starting the container main process. The file is deleted immediately after sourcing.
5. Remove all env variables from the K8s manifest.

## Config Limitations

- Only works on Deployments, StatefulSets and Daemonsets during creation and update.
- All env variables have to be written straight into the K8s manifests. Reading from Secrets or ConfigMaps is currently not supported.
- Entrypoint and / or command of application container image has to be overwritten in K8s manifest with command and / or args.

## Security Limitations

- Environment values will still be visible from host machine proc file system.
- Environment values will still be visible in K8s manifest.

## Installation

### Prerequisits

Internal K8s traffic between infrastructure components is mandated to use TLS - therefore your need:

- A server private key PEM file.
- A certificate signing request CSR file.

### Obtain a signed certificate

#### Step 1 - Create K8s CertificateSigningRequest:
```
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
```
kubectl certificate approve webhook.default
```
#### Step 3 - Download the approved certificate:
```
kubectl get csr webhook.default -o jsonpath='{.status.certificate}' \
| base64 --decode > server.crt
```
#### Step 4 - Install the certificate as a TLS secret in your K8s cluster:
```
kubectl create secret tls webhook-tls --cert=server.crt --key=server-key.pem
```

### Install K8s Hide Env
```
kubectl apply -f src/k8s/service.yaml
kubectl apply -f src/k8s/k8s-hide-env-deployment.yaml
```

### Create the Webhook

Please note that here you have to provide the K8s cluster root certificate to validate TLS connections. E.g. with Minikube, this can be done as follows:

```
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
