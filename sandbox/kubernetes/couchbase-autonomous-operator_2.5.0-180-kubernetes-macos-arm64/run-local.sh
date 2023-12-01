#!/bin/bash
kubectl apply -f crd.yaml  
bin/cao create admission   
bin/cao create operator
kubectl apply -f couchbase-cluster.yaml
kubectl get pods
# wait for pods to be ready
kubectl port-forward cb-example-0000 8091