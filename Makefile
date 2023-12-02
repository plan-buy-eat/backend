SERVICE_VERSION = 0.0.2

.PHONY: docker-build-bastion
docker-build-bastion:
	docker build -t oltur/bastion:latest devops/docker -f devops/docker/bastion.Dockerfile
	docker push oltur/bastion:latest

.PHONY: kube-deploy-bastion
kube-deploy-bastion: docker-build-bastion
	# kind load docker-image bastion:
	kubectl apply -f devops/k8s/pod-bastion.yaml

.PHONY: docker-build-user-service
docker-build-user-service:
	docker build -t oltur/user-service:$(SERVICE_VERSION) src/user-service
	docker push oltur/user-service:$(SERVICE_VERSION)

.PHONY: docker-build-user-service-debug
docker-build-user-service-debug:
	docker build -t oltur/user-service:$(SERVICE_VERSION) src/user-service -f src/user-service/debug.Dockerfile
	docker push oltur/user-service:$(SERVICE_VERSION)

.PHONY: kube-deploy-user-service
kube-deploy-user-service: docker-build-user-service
	# kind load docker-image user-service:
	SERVICE_VERSION=$(SERVICE_VERSION) envsubst < devops/k8s/deployment-user-service.yaml | kubectl apply -f -
	kubectl apply -f devops/k8s/service-user-service.yaml
	kubectl apply -f devops/k8s/ingress-user-service.yaml

.PHONY: kube-deploy-user-service-debug
kube-deploy-user-service-debug: docker-build-user-service-debug
	# kind load docker-image user-service:
	SERVICE_VERSION=$(SERVICE_VERSION) envsubst < devops/k8s/deployment-user-service-debug.yaml | kubectl apply -f -
	kubectl apply -f devops/k8s/service-user-service.yaml
	kubectl apply -f devops/k8s/ingress-user-service.yaml

.PHONY: kube-delete-user-service
kube-delete-user-service:
	kubectl delete -f devops/k8s/ingress.yaml
	kubectl delete -f devops/k8s/service.yaml
	kubectl delete deployment service-app

.PHONY: helm-deploy-couchbase
helm-deploy-couchbase:
	helm repo add couchbase https://couchbase-partners.github.io/helm-charts/
	helm repo update
	helm install couchbase --set cluster.name=couchbase --values devops/helm/values.yaml --set tls.generate=true couchbase/couchbase-operator

.PHONY: helm-delete-couchbase
helm-delete-couchbase:
	helm uninstall couchbase
