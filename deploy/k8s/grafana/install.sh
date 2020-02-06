kubectl apply -f namespace.yml
helm install --namespace grafana grafana stable/grafana -f grafana.yml