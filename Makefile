SERVICE_VERSION = 0.0.11

all:
	$(eval GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD))
	echo "Git branch is $(GIT_BRANCH)"

.PHONY: build-bastion
build-bastion:
	docker build -t oltur/bastion:latest devops/docker -f devops/docker/bastion.Dockerfile
	docker push oltur/bastion:latest

.PHONY: install-bastion
install-bastion: # build-bastion
	helm upgrade --install bastion-host --values devops/bastion-host/values.yaml  ./devops/bastion-host

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
	docker build -t oltur/user-service:$(SERVICE_VERSION) --build-arg SERVICE_NAME=user-service -f src/user-service/Dockerfile ./src
	docker push oltur/user-service:$(SERVICE_VERSION)

.PHONY: build-user-service-debug
build-user-service-debug:
	docker build -t oltur/user-service:debug-$(SERVICE_VERSION)  --build-arg SERVICE_NAME=user-service -f src/user-service/Dockerfile ./src
	docker push oltur/user-service:debug-$(SERVICE_VERSION)

#.PHONY: install-user-service
#install-user-service: # build-user-service
#	# kind load docker-image user-service:
#	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase --namespace couchbase | sed -n -e 's/^.*password: //p'))
#	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
#	$(eval DEBUG=false)
#	helm upgrade --install user-service --values devops/user-service/values.yaml --set SERVICE_NAME=user-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: upgrade-user-service
upgrade-user-service: # build-user-service
	# kind load docker-image user-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase --namespace couchbase | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=false)
	helm upgrade --install user-service --values devops/user-service/values.yaml --set SERVICE_NAME=user-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

#.PHONY: install-user-service-debug
#install-user-service-debug: # build-user-service-debug
#	# kind load docker-image user-service:
#	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase --namespace couchbase | sed -n -e 's/^.*password: //p'))
#	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
#	$(eval DEBUG=true)
#	helm upgrade --install user-service --values devops/user-service/values.yaml --set SERVICE_NAME=user-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: upgrade-user-service-debug
upgrade-user-service-debug: # build-user-service-debug
	# kind load docker-image user-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase --namespace couchbase | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=true)
	helm upgrade --install user-service --values devops/user-service/values.yaml --set SERVICE_NAME=user-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: uninstall-user-service
uninstall-user-service:
	helm uninstall user-service

#==================================================================================================

.PHONY: build-item-service
build-item-service:
	docker build -t oltur/item-service:$(SERVICE_VERSION) --build-arg SERVICE_NAME=user-service -f src/item-service/Dockerfile ./src
	docker push oltur/item-service:$(SERVICE_VERSION)

.PHONY: build-item-service-debug
build-item-service-debug:
	docker build -t oltur/item-service:debug-$(SERVICE_VERSION) --build-arg SERVICE_NAME=user-service -f src/item-service/debug.Dockerfile ./src
	docker push oltur/item-service:debug-$(SERVICE_VERSION)

#.PHONY: install-item-service
#install-item-service: # build-item-service
#	# kind load docker-image item-service:
#	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase --namespace couchbase | sed -n -e 's/^.*password: //p'))
#	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
#	$(eval DEBUG=false)
#	helm upgrade --install item-service --values devops/item-service/values.yaml --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

#.PHONY: install-item-service-debug
#install-item-service-debug: # build-item-service-debug
#	# kind load docker-image item-service:
#	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase --namespace couchbase | sed -n -e 's/^.*password: //p'))
#	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
#	$(eval DEBUG=true)
#	helm upgrade --install item-service --values devops/item-service/values.yaml --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: upgrade-item-service
upgrade-item-service: # build-item-service
	# kind load docker-image item-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase --namespace couchbase | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=false)
	helm upgrade --install item-service --values devops/item-service/values.yaml --set SERVICE_NAME=item-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: upgrade-item-service-debug
upgrade-item-service-debug: # build-item-service-debug
	# kind load docker-image item-service:
	$(eval COUCHBASE_PASSWORD=$(shell helm status couchbase --namespace couchbase | sed -n -e 's/^.*password: //p'))
	$(eval SERVICE_VERSION=$(SERVICE_VERSION))
	$(eval DEBUG=true)
	helm upgrade --install item-service --values devops/item-service/values.yaml --set SERVICE_NAME=item-service --set DEBUG=$(DEBUG) --set COUCHBASE_PASSWORD=$(COUCHBASE_PASSWORD) --set SERVICE_VERSION=$(SERVICE_VERSION) devops/service

.PHONY: uninstall-item-service
uninstall-item-service:
	helm uninstall item-service

#====================================

.PHONY: install-kube-prometheus-stack
install-kube-prometheus-stack:
	helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace --values devops/kube-prometheus-stack/values.yaml

.PHONY: uninstall-kube-prometheus-stack
uninstall-kube-prometheus-stack:
	helm uninstall kube-prometheus-stack --namespace monitoring

.PHONY: expose-kube-prometheus-stack-prometheus
expose-kube-prometheus-stack-prometheus:
	kubectl --namespace monitoring port-forward svc/kube-prometheus-stack 9090
	# Then access via http://localhost:9090

.PHONY: expose-kube-prometheus-stack-grafana
expose-kube-prometheus-stack-grafana:
	kubectl --namespace monitoring port-forward svc/kube-prometheus-stack-grafana 3000:80
	# Then access via http://localhost:3000

.PHONY: expose-kube-prometheus-stack-alertmanager
expose-kube-prometheus-stack-alertmanager:
	kubectl --namespace monitoring port-forward svc/alertmanager-main 9093
	# Then access via http://localhost:9093

.PHONY: expose-loki
expose-loki:
	kubectl port-forward svc/loki 3000:80
	# Then access via http://localhost:3000

.PHONY: expose-zipkin
expose-zipkin:
	kubectl port-forward svc/zipkin 9411:9411
	# Then access via http://localhost:9411
#==================================================================================================

.PHONY: install-traefik
install-traefik:
	helm repo add traefik https://traefik.github.io/charts
	helm upgrade --install traefik traefik/traefik --values devops/k8s/traefik/values.yaml

.PHONY: uninstall-traefik
uninstall-traefik:
	helm uninstall traefik

.PHONY: expose-traefik-dashboard
expose-traefik-dashboard:
	kubectl port-forward $(kubectl get pods --selector "app.kubernetes.io/name=traefik" --output=name) 9000:9000

#==================================================================================================

.PHONY: install-couchbase-local
install-couchbase-local:
	docker run -d --name couchbase -p 8091-8094:8091-8094 -p 11210:11210 couchbase

.PHONY: install-couchbase
install-couchbase:
	helm repo add couchbase https://couchbase-partners.github.io/helm-charts/
	helm repo update
#	helm upgrade --install couchbase --set cluster.name=couchbase --values devops/couchbase/values.yaml --set tls.generate=true couchbase/couchbase-operator
#	helm upgrade --install couchbase --set cluster.name=couchbase --values devops/couchbase/values.yaml --set tls.generate=true ./devops/couchbase/couchbase-operator
	helm upgrade --install couchbase --set cluster.name=couchbase --values devops/couchbase/values.yaml --namespace couchbase --create-namespace ./devops/couchbase/couchbase-operator

.PHONY: uninstall-couchbase
uninstall-couchbase:
	helm uninstall couchbase --namespace couchbase

.PHONY: status-couchbase
status-couchbase:
	helm status couchbase --namespace couchbase

.PHONY: expose-couchbase-ui
expose-couchbase-ui:
	kubectl port-forward --namespace couchbase service/couchbase 38091:8091

.PHONY: expose-couchbase-api
expose-couchbase-api:
	kubectl port-forward --namespace couchbase couchbase-0000 38094:8094

#==================================================================================================

.PHONY: install-test-service
install-test-service:
	helm upgrade --install test-service --values devops/test-service/values.yaml  ./devops/test-service # --namespace test

.PHONY: uninstall-test-service
uninstall-test-service:
	helm uninstall test-service

.PHONY: install-ingress-example
install-ingress-example:
	helm upgrade --install ingress-example ./devops/kind/ingress-example

.PHONY: uninstall-ingress-example
uninstall-ingress-example:
	helm uninstall ingress-example

#==================================================================================================
.PHONY: lint
lint:
	$(eval FILTER_REGEX_EXCLUDE=".*(sandbox|couchbase|templates|devops/otel/local).*")
	echo $(FILTER_REGEX_EXCLUDE)
	docker run \
      -e RUN_LOCAL=true \
      -e VALIDATE_GO=false \
      -e IGNORE_GITIGNORED_FILES=true \
	  -e BASH_SEVERITY=error \
      -e VALIDATE_NATURAL_LANGUAGE=false \
      -e VALIDATE_DOCKERFILE_HADOLINT=false \
      -e VALIDATE_ENV=false \
      -e VALIDATE_JSCPD=false \
      -e FILTER_REGEX_EXCLUDE=$(FILTER_REGEX_EXCLUDE) \
      -e KUBERNETES_KUBECONFORM_OPTIONS="--ignore-missing-schemas" \
      -e LINTER_RULES_PATH=".github/linters" \
      -e GITHUB_ACTIONS_COMMAND_ARGS="-shellcheck=" \
      -v .:/tmp/lint \
      --rm \
      --platform linux/amd64 \
      ghcr.io/super-linter/super-linter:latest
#            -e ACTIONS_RUNNER_DEBUG=true \

.PHONY: install-self-hosted-github-runners
install-self-hosted-github-runners:
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.crds.yaml
	helm repo add jetstack https://charts.jetstack.io
	helm upgrade --install cert-manager --namespace cert-manager --version v1.13.3 jetstack/cert-manager --create-namespace
	kubectl create namespace actions-runner-system --dry-run=client -o yaml | kubectl apply -f -
	kubectl create secret generic controller-manager -n actions-runner-system --from-literal=github_token=ghp_?????????????????? #self-hosted-runners-cluster-prod
	helm repo add actions-runner-controller https://actions-runner-controller.github.io/actions-runner-controller
	helm repo update
	helm upgrade --install --namespace actions-runner-system --create-namespace --wait actions-runner-controller actions-runner-controller/actions-runner-controller --set syncPeriod=1m
	kubectl create -f ./devops/self-hosted-runners/runner.yaml
	kubectl apply -f ./devops/self-hosted-runners/horizontal_runner_autoscaler.yaml

.PHONY: install-otel-collector
install-otel-collector:
	helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
	helm upgrade --install my-opentelemetry-collector open-telemetry/opentelemetry-collector --set mode=deployment --values devops/otel/opentelemetry-collector/values.yaml
	# --set mode=<daemonset|deployment|statefulset>

.PHONY: install-loki
install-loki:
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	
	k apply -f devops/loki/sc.yaml   
	kubectl patch storageclass local-storage -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
	k apply -f devops/loki/pv.yaml 
	helm upgrade --install loki grafana/loki --values devops/loki/values.yaml 

.PHONY: uninstall-loki
uninstall-loki:
	helm uninstall loki

.PHONY: install-zipkin
install-zipkin:
	# helm repo add zipkin-helm https://financial-times.github.io/zipkin-helm/docs
	# helm upgrade --install zipkin-helm zipkin-helm/zipkin-helm
	k apply -f devops/zipkin/zipkin.yaml

.PHONY: uninstall-zipkin
uninstall-zipkin:
	# helm uninstall zipkin-helm 
	k delete -f devops/zipkin/zipkin.yaml

# .PHONY: install-otel-collector
# install-otel-collector:
# 	# kubectl apply -f ./devops/otel/collector.yaml
# 	kubectl apply -f ./devops/otel/collector/otel-config.yaml


# now using one above
# .PHONY: uninstall-otel-collector
# uninstall-otel-collector:
# 	# kubectl delete -f ./devops/otel/collector.yaml
# 	kubectl delete -f ./devops/otel/collector/otel-config.yaml

.PHONY: install-local-otel-collector
install-local-otel-collector:
	docker pull otel/opentelemetry-collector:0.93.0
	docker run -d --name otel-collector -p 127.0.0.1:4317:4317 -p 127.0.0.1:55679:55679 otel/opentelemetry-collector:0.93.0

.PHONY: install-metrics-server
install-metrics-server:
	kubectl apply -f ./devops/metrics-server/components.yaml

# .PHONY: install-k8s-otel-collector
# install-k8s-otel-collector:
# 	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
# 	helm repo update
# 	helm upgrade --install ksm prometheus-community/kube-state-metrics -n "default"
# 	helm upgrade --install nodeexporter prometheus-community/prometheus-node-exporter -n "default"
# 	TBD

# .PHONY: install-prometheus
# install-prometheus:
# 	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts