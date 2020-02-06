kubectl apply -f namespace.yml
kubectl apply -f storageclass.yml
helm install -namespace selfops postgresql stable/postgresql -f postgres.yml
