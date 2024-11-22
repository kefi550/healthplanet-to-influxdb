
```
kubectl create ns healthplanet
# sealed-secretが必要
kubectl apply -f secret-sealed.yaml
kubectl apply -f cronjob.yaml
```
