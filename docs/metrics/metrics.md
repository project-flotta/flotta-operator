# Publication of the metrics in Prometheus on OCP

In order to publish the metrics several steps need to be done:

- Enable monitoring on the cluster
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
data:
  config.yaml: |
EOF
``` 
