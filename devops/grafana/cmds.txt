#!/bin/sh
# helm repo add stable https://charts.helm.sh/stable
# helm repo add prometheus-community https://prometheus-community.github.io/helm-chart
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace
# -f values.yaml
