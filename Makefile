install_certmanager:
	# install cert-manager 0.16.1
	helm repo add jetstack https://charts.jetstack.io
	helm repo update
	helm install cert-manager jetstack/cert-manager \
	  --namespace cert-manager \
	  --version v0.16.1 \
	  --set installCRDs=true \
	  --create-namespace=true

create_selfsigned_cert:
	# install self-signed cluster issuer
	kubectl apply -f deployments/clusterissuer.yaml
	# install self-signed certificate
	kubectl apply -f deployments/certificate.yaml

install_k8shideenv_deployment:
	kubectl apply -f deployments/serviceaccount.yaml
	kubectl apply -f deployments/clusterrole.yaml
	kubectl apply -f deployments/clusterrolebinding.yaml
	kubectl apply -f deployments/service.yaml
	kubectl apply -f deployments/deployment.yaml

delete_k8shideenv_deployment:
	kubectl delete mutatingwebhookconfigurations k8s-hide-env
	kubectl delete deploy k8s-hide-env
	kubectl delete svc k8s-hide-env

delete_selfsigned_cert:
	kubectl delete secret k8s-hide-env-tls

build_image:
	docker build -t k8s-hide-env .
