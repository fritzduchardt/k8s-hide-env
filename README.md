# K8s Hide Env

K8s Hide Env removes environment variables from your container environment and makes them  **only visible to the container main process and its child processes**.

By default, K8s exposes environment variables in the container environment. This carries **substantial security risks**, since it allows container hijackers to gain access to infrastructure credentials and increases the risk for credentials exposure with lifecycle hooks.

## How it works

K8s Hide Env installs a [Mutating Web Hook](https://kubernetes.io/blog/2019/03/21/a-guide-to-kubernetes-admission-controllers/) in your K8s cluster, which in a nut-shell **extracts your container environment variables from the K8s manifest and adds them exclusively to the shell that starts the main process by amending container *command* and *args*.** Then environment variables are deleted from K8s manifest.

## Limitations

- Only works on *Deployments*, *StatefulSets* and *Daemonsets*.
- All environment variables have to be written straight into the K8s manifests. Reading from *Secrets* or *ConfigMaps* is currently not supported.
- `ENTRYPOINT` and / or `CMD` configuration of the application container image has to get overwritten in K8s manifest with the `command` and / or `args` element.
- Environment values will still be visible from the worker node proc file system.
- Environment values will still be visible in the K8s manifest.

## Installation

### Obtain a signed certificate

For detailed information about how to create a self-signed TLS certificate in K8s refer to this [documentation](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/).

Alternatively, you can use [cert-manager](https://cert-manager.io/) which is a bit easier. Below we describe how to go ahead with cert-manager and a [SelfSigned Issuer](https://cert-manager.io/docs/configuration/selfsigned/). 

#### Create a Self-Signed Certificate with CertManager like that:
```shell
kubectl create -f -<<EOF
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: k8s-hide-env
  namespace: default
spec:
  # Secret names are always required.
  secretName: k8s-hide-env-tls
  duration: 2160h # 90d
  renewBefore: 360h # 15d
  subject:
    organizations:
    - github
  isCA: false
  privateKey:
    algorithm: RSA
    encoding: PKCS1
    size: 2048
  usages:
    - server auth
    - client auth
  dnsNames:
  - k8s-hide-env.default.svc.cluster.local
  - k8s-hide-env.default.svc
  - k8s-hide-env.default
  - k8s-hide-env
  issuerRef:
    name: selfsigned-issuer
    kind: Issuer
EOF
```
### Install K8s Hide Env
```shell
kubectl apply -f deployment/service.yaml
kubectl apply -f deployment/deployment.yaml
```

### Create the Webhook

Please note that here you have to provide the root certificate your server certs were signed with to validate TLS connections. Since in this Howto, we let K8s do the signing with the Minikube root certificate, let's provide the Minikube root certificate to the web hook configuration:

```shell
cat <<EOF | kubectl replace --force -f -
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: k8s-hide-env
  labels:
    app: k8s-hide-env
webhooks:
  - name: k8s-hide-env.default.svc.cluster.local
    sideEffects: None
    admissionReviewVersions: ["v1", "v1beta1"]
    matchPolicy: Equivalent
    objectSelector:
      matchLabels:
        mode: secure
    clientConfig:
      caBundle: $(kubectl get secret k8s-hide-env-tls -o jsonpath='{.data.ca\.crt}')
      service:
        name: k8s-hide-env
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
kubectl apply -f test/deploy.yaml
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
## Uninstall

The following commands will remove all traces of K8s-hide-env from your cluster: 

```
kubectl delete mutatingwebhookconfigurations k8s-hide-env
kubectl delete deploy k8s-hide-env
kubectl delete svc k8s-hide-env
#kubectl delete secret k8s-hide-env-tls
```

## Feedback

Any ideas and / or feedback regarding this project is very welcome. Please write to [fritz@duchardt.net](mailto:fritz@duchardt.net).