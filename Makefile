install_certmanager:
	helm repo add jetstack https://charts.jetstack.io
	helm repo update
	helm install cert-manager jetstack/cert-manager \
	  --namespace cert-manager \
	  --set installCRDs=true \
	  --create-namespace=true

install_k8shideenv_deployment:
	helm install k8s-hide-env charts/k8s-hide-env

delete_k8shideenv_deployment:
	helm delete k8s-hide-env

build_image:
	docker build -t k8s-hide-env .
