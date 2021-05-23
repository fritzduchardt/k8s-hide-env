# K8s Hide Env

K8s Hide Env removes environment variables from your container environment and makes them  **only visible to the container main process and its child processes**.

By default, K8s exposes environment variables in the container environment. This carries **substantial security risks**, since it allows container hijackers to gain access to infrastructure credentials and increases the risk for credentials exposure with lifecycle hooks.

## How it works

K8s Hide Env installs a [Mutating Web Hook](https://kubernetes.io/blog/2019/03/21/a-guide-to-kubernetes-admission-controllers/) in your K8s cluster, which in a nut-shell **extracts your container environment variables from the K8s manifest and adds them to a K8s secret. On startup, they are made available exclusively to the shell that starts the main process by amending container *command* and *args*.** Then environment variables are deleted from K8s manifest.

## Limitations

- Only works on *Deployments*, *StatefulSets* and *Daemonsets*.
- All environment variables have to be written straight into the K8s manifests. Reading from *Secrets* or *ConfigMaps* is currently not supported.
- `ENTRYPOINT` and / or `CMD` configuration of the application container image has to get overwritten in K8s manifest with the `command` and / or `args` element.
- Environment values will still be visible from the worker node proc file system.

## Installation

### Installing Cert-Manager

K8s internal communication is mandated to use TLS. Therefore, Mutating Web Hook applications need to expose ports that accept TLS traffic. Also, the corresponding Mutating Web Hook Configuration needs to include the CA cert that was used to create the application certificates.

We use [cert-manager](https://cert-manager.io/) for the certificate creation and renewal. We use a [SelfSigned Issuer](https://cert-manager.io/docs/configuration/selfsigned/) in order to create the application certificate and [Cainjector](https://cert-manager.io/docs/concepts/ca-injector/) to provide the CA certificate to the Mutating Web Hook Configuration.

#### Install Cert-Manager including CAInjector
```shell
make install_certmanager
```

### Install K8s Hide Env including everything (Mutating Web Hook Configuration, Certificate Issuer, Certificate, Application, RBAC configuration)

We are using Helm in order to do the installation. The corresponding chart can be found under `./charts/k8s-hide-env`.

```shell
make install_k8shideenv
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
> ICanSeeYou
```
## Uninstall

The following commands will remove all traces of K8s-hide-env from your cluster: 

```
make delete_k8shideenv_deployment
```

## Feedback

Any ideas and / or feedback regarding this project is very welcome. Please write to [fritz@duchardt.net](mailto:fritz@duchardt.net).