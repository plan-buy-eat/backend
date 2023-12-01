SERVICE_VERSION = 4

.PHONY: docker-build
docker-build:
	docker build -t service1:$(SERVICE_VERSION) src/service
	docker tag service1:$(SERVICE_VERSION) oltur/service1:$(SERVICE_VERSION)
	docker tag service1:$(SERVICE_VERSION) oltur/service1:$(SERVICE_VERSION)
	docker  push  oltur/service1:$(SERVICE_VERSION)

.PHONY: kube-apply
kube-apply: docker-build
	cd src/service
	# kind load docker-image service1:latest
	SERVICE_VERSION=$(SERVICE_VERSION) envsubst < devops/k8s/deployment.yaml | kubectl apply -f -
	kubectl apply -f devops/k8s/service.yaml
	kubectl apply -f devops/k8s/ingress.yaml


.PHONY: kube-delete
kube-delete:
	kubectl delete -f devops/k8s/ingress.yaml
	kubectl delete -f devops/k8s/service.yaml
	kubectl delete deployment service-app

.PHONY: helm-apply
helm-apply:
	helm repo add couchbase https://couchbase-partners.github.io/helm-charts/
	helm repo update
	helm install couchbase --set cluster.name=couchbase --values devops/helm/values.yaml --set tls.generate=true couchbase/couchbase-operator
