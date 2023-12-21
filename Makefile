SERVICE_VERSION = 0.0.7

all:
	$(eval GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD))
	echo "Git branch is $(GIT_BRANCH)"

.PHONY: build-bastion
build-bastion:
	docker build -t oltur/bastion:latest devops/docker -f devops/docker/bastion.Dockerfile
	docker push oltur/bastion:latest

.PHONY:install-bastion
install-bastion: #build-bastion
	helm install bastion-host --values devops/bastion-host/values.yaml  ./devops/bastion-host

.PHONY:uninstall-bastion
uninstall-bastion: #build-bastion
	helm uninstall bastion-host

#==================================================================================================

.PHONY: build-all
build-all: build-item-service build-user-service

.PHONY: build-all-debug
build-all-debug: build-item-service-debug build-user-service-debug

.PHONY: upgrade-all
upgrade-all: upgrade-item-service upgrade-user-service

.PHONY: upgrade-all-debug
upgrade-all-debug: upgrade-item-service-debug upgrade-user-service-debug

#==================================================================================================

.PHONY: build-user-service
build-user-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	docker build -t oltur/user-service:$(SERVICE_VERSION) --build-arg SERVICE_NAME=user-service --build-arg COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) -f src/user-service/Dockerfile ./src
	docker push oltur/user-service:$(SERVICE_VERSION)

.PHONY: build-user-service-debug
build-user-service-debug:
	docker build -t oltur/user-service:debug-$(SERVICE_VERSION)  --build-arg SERVICE_NAME=user-service --build-arg COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) -f src/user-service/Dockerfile ./src
	docker push oltur/user-service:debug-$(SERVICE_VERSION)

.PHONY: install-user-service
install-user-service: # build-user-service
	# kind load docker-image user-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=false)
	helm install user-service --values devops/user-service/values.yaml --set SERVICE_NAME=user-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: upgrade-user-service
upgrade-user-service: # build-user-service
	# kind load docker-image user-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=false)
	helm upgrade user-service --values devops/user-service/values.yaml --set SERVICE_NAME=user-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: install-user-service-debug
install-user-service-debug: # build-user-service-debug
	# kind load docker-image user-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=true)
	helm install user-service --values devops/user-service/values.yaml --set SERVICE_NAME=user-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: upgrade-user-service-debug
upgrade-user-service-debug: # build-user-service-debug
	# kind load docker-image user-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=true)
	helm upgrade user-service --values devops/user-service/values.yaml --set SERVICE_NAME=user-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: uninstall-user-service
uninstall-user-service:
	helm uninstall user-service

#==================================================================================================

.PHONY: build-item-service
build-item-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	docker build -t oltur/item-service:$(SERVICE_VERSION) --build-arg SERVICE_NAME=user-service --build-arg COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) -f src/item-service/Dockerfile ./src
	docker push oltur/item-service:$(SERVICE_VERSION)

.PHONY: build-item-service-debug
build-item-service-debug:
	docker build -t oltur/item-service:debug-$(SERVICE_VERSION) --build-arg SERVICE_NAME=user-service --build-arg COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) -f src/item-service/debug.Dockerfile ./src
	docker push oltur/item-service:debug-$(SERVICE_VERSION)

.PHONY: install-item-service
install-item-service: # build-item-service
	# kind load docker-image item-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=false)
	helm install item-service --values devops/item-service/values.yaml --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: install-item-service-debug
install-item-service-debug: # build-item-service-debug
	# kind load docker-image item-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=true)
	helm install item-service --values devops/item-service/values.yaml --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: upgrade-item-service
upgrade-item-service: # build-item-service
	# kind load docker-image item-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=false)
	helm upgrade item-service --values devops/item-service/values.yaml --set SERVICE_NAME=item-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: upgrade-item-service-debug
upgrade-item-service-debug: # build-item-service-debug
	# kind load docker-image item-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase  | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=true)
	helm upgrade item-service --values devops/item-service/values.yaml --set SERVICE_NAME=item-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: uninstall-item-service
uninstall-item-service:
	helm uninstall item-service

#==================================================================================================
.PHONY: install-couchbase-local
install-couchbase-local:
	docker run -d --name couchbase -p 8091-8094:8091-8094 -p 11210:11210 couchbase

.PHONY: install-couchbase
install-couchbase:
#	helm repo add couchbase https://couchbase-partners.github.io/helm-charts/
#	helm repo update
#	helm install couchbase --set cluster.name=couchbase --values devops/couchbase/values.yaml --set tls.generate=true couchbase/couchbase-operator
	helm install couchbase --set cluster.name=couchbase --values devops/couchbase/values.yaml --set tls.generate=true ./devops/couchbase/couchbase-operator

.PHONY: uninstall-couchbase
uninstall-couchbase:
	helm uninstall couchbase

.PHONY: status-couchbase
status-couchbase:
	helm status couchbase

.PHONY: install-test-service
install-test-service:
	helm install test-service --values devops/test-service/values.yaml  ./devops/test-service # --namespace test

.PHONY: uninstall-test-service
uninstall-test-service:
	helm uninstall test-service

.PHONY: create-kind-cluster
create-kind-cluster:
	kind create cluster --config ./devops/kind/kind-config.yaml

.PHONY: create-kind-ingress
create-kind-ingress:
#	kubectl apply -f ./devops/kind/contour.yaml
#	kubectl patch daemonsets -n projectcontour envoy -p '{"spec":{"template":{"spec":{"nodeSelector":{"ingress-ready":"true"},"tolerations":[{"key":"node-role.kubernetes.io/control-plane","operator":"Equal","effect":"NoSchedule"},{"key":"node-role.kubernetes.io/master","operator":"Equal","effect":"NoSchedule"}]}}}}'
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
	kubectl wait --namespace ingress-nginx \
      --for=condition=ready pod \
      --selector=app.kubernetes.io/component=controller \
      --timeout=90s

.PHONY: delete-kind-cluster
delete-kind-cluster:
	cd devops/kind && kind delete cluster

.PHONY: install-ingress-example
install-ingress-example:
	helm install ingress-example ./devops/kind/ingress-example

.PHONY: uninstall-ingress-example
uninstall-ingress-example:
	helm uninstall ingress-example