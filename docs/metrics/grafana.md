# Building Grafana Dashboard

- In order to install Grafana in flotta namespace, the use the following script which will install the Grafana and Grafana Dashboard.
```
export KUBECONFIG=your-kubeconfig-file
tools/deploy_grafana.sh -d docs/metrics/flotta-dashboard.json
```

- To import any additional Grafana dashboard to existing Grafana in flotta namespace, use following script:
  ```shell
  export KUBECONFIG=your-kubeconfig-file
  tools/import_grafana_dashboards.sh -d <dashboard file>
  ```
  Specifically, it can be used to install edge device health monitoring dashboard (docs/metrics/flotta-devices-health.json):
  ```shell
  tools/import_grafana_dashboards.sh -d docs/metrics/flotta-devices-health.json
  ```
