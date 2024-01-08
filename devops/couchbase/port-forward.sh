#!/bin/sh
kubectl port-forward --namespace default couchbase-0000 18091:18091 &
