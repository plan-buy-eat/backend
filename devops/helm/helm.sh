helm repo add couchbase https://couchbase-partners.github.io/helm-charts/
helm repo update
helm install couchbase --set cluster.name=couchbase --values values.yaml --set tls.generate=true couchbase/couchbase-operator