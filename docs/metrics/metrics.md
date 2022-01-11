# Publication of the metrics

In order to publish the metrics several steps need to be done:
- Create a service monitor
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: k4e-operator-servicemonitor
  namespace: k4e-operator-system
  labels:
    control-plane: controller-manager
spec:
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
  selector:
    matchLabels:
      control-plane: controller-manager
```
- Add port 8080 to k4e-operator-controller-manager service
```yaml
    - name: metrics
      protocol: TCP
      port: 8080
      targetPort: 8080
```
- Enable user-defined for user projects
```yaml
apiVersion: v1
data:
  config.yaml: |
    enableUserWorkload: true
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
``` 
