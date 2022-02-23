#!/bin/sh

# This script deploys Grafana on the cluster and import the flotta dashboard.
# - the first parameter is the dashboard file to import

# check if arguments are passed
if [ $# -eq 0 ]; then
  DASHBOARD_FILE="./docs/metrics/flotta-dashboard.json"
  echo "No arguments supplied. Using default dashboard $DASHBOARD_FILE."
else
  DASHBOARD_FILE=$1
fi

if [ ! -f "$1" ]; then
  echo "File $1 does not exist"
  exit 1
fi

# Deploy Grafana operator
kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: grafana-operator
  namespace: flotta
spec:
  targetNamespaces:
    - flotta
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  labels:
    operators.coreos.com/grafana-operator.flotta: ''
  name: grafana-operator
  namespace: flotta
spec:
  channel: v4
  installPlanApproval: Automatic
  name: grafana-operator
  source: community-operators
  sourceNamespace: openshift-marketplace
  startingCSV: grafana-operator.v4.2.0
EOF

kubectl wait subscription -n flotta grafana-operator --for condition=CatalogSourcesUnhealthy=False --timeout=60s
sleep 10
kubectl wait deployment -n flotta -l operators.coreos.com/grafana-operator.flotta= --for condition=Available=True --timeout=60s

# Create Grafana instance
kubectl apply -f - <<EOF
apiVersion: integreatly.org/v1alpha1
kind: Grafana
metadata:
 name: grafana
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
EOF
kubectl wait deployment -n flotta grafana-deployment --for condition=Available=True --timeout=90s
kubectl wait pod -n flotta -lapp=grafana --for condition=READY=True --timeout=90s

oc adm policy add-cluster-role-to-user cluster-monitoring-view -z grafana-serviceaccount -n flotta
BEARER_TOKEN=$(oc serviceaccounts get-token grafana-serviceaccount -n flotta)

# Create Grafana datasource
kubectl apply -f - <<EOF
apiVersion: integreatly.org/v1alpha1
kind: GrafanaDataSource
metadata:
  name: flotta-datasource
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
EOF

sleep 5
GRAFANA_API="https://root:secret@$(oc get routes -n flotta grafana-route --no-headers -o=custom-columns=HOST:.spec.host)/api/dashboards/import"

request_body=$(mktemp)
cat <<EOF >> $request_body
{
  "dashboard": $(cat $DASHBOARD_FILE),
  "folderId": 0,
  "overwrite": true
}
EOF

# Import flotta dashboard
curl -X POST --insecure -H "Content-Type: application/json" -d @$request_body "$GRAFANA_API"

# Clean up of grafana resource in flotta namespace
uninstall_grafana() {
    kubectl delete grafanadatasource -n flotta flotta-datasource
    kubectl delete grafana -n flotta grafana
    kubectl delete subscription -n flotta grafana-operator
    kubectl delete operatorgroup -n flotta grafana-operator
}