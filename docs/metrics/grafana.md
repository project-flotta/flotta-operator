# Building Grafana Dashboard

- In order to install Grafana in flotta namespace, the use the following script which will install the Grafana and Grafana Dashboard.
```
export KUBECONFIG=your-kubeconfig-file
tools/install-grafana.sh docs/metrics/flotta-dashboard.yaml
```