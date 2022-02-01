# Building Grafana Dashboard

-  Install Grafana from the operatorHub on `flotta` namespace
-  Create a Gafana instance
 ```yaml
apiVersion: integreatly.org/v1alpha1
kind: Grafana
metadata:
  name: grafana-example
  namespace: flotta
spec:
  config:
    auth:
      disable_signout_menu: true
    auth.anonymous: {}
    security:
      admin_password: secret
      admin_user: root
  ingress:
    enabled: true
 ``` 
-  Connecting Prometheus to our Custom Grafana
    -  Grant cluster-monitoring-view cluster role to the  grafana-serviceaccount service account, that was created alongside the Grafana instance\
       `oc adm policy add-cluster-role-to-user cluster-monitoring-view -z grafana-serviceaccount -n flotta`
    -  Generate Bearer Token:\
       `oc serviceaccounts get-token grafana-serviceaccount -n flotta`
    -  Create Grafana Data Source resource and replace the `${BEARER_TOKEN}` with the output of the previous command:\
```yaml
apiVersion: integreatly.org/v1alpha1
kind: GrafanaDataSource
metadata:
  name: prometheus-grafanadatasource
  namespace: flotta
spec:
  datasources:
    - access: proxy
      editable: true
      isDefault: true
      jsonData:
        httpHeaderName1: 'Authorization'
        timeInterval: 5s
        tlsSkipVerify: true
      name: Prometheus
      secureJsonData:
        httpHeaderValue1: 'Bearer ${BEARER_TOKEN}'
      type: prometheus
      url: 'https://thanos-querier.openshift-monitoring.svc.cluster.local:9091'
  name: prometheus-grafanadatasource.yaml
```    
-  Log into Grafana page:
    -  Networking -> Routes -> select grafana-route -> use the link of 'Location'
    -  Use user `root` and password `secret` (that configured in grafana instance)
-  Import the dashboard json
    -  Click on the `+` button and choose `import"
    -  Upload the json named `flotta-dashboard.json`
 