#!/bin/sh

usage()
{
cat << EOF
usage: $0 options
This script will deploy Grafana on the cluster and import the flotta dashboard.
OPTIONS:
   -h      Show this message
   -d      The dashboard to import
   -u      Uninstall Grafana
EOF
}

# Clean up of grafana resource in flotta namespace
uninstall_grafana() {
    kubectl delete grafanadatasource -n flotta flotta-datasource
    kubectl delete grafana -n flotta grafana
    kubectl delete clusterserviceversion -n flotta -l operators.coreos.com/grafana-operator.flotta=
    kubectl delete subscription -n flotta grafana-operator
    kubectl delete operatorgroup -n flotta grafana-operator
}

while getopts "h:d:u" option; do
    case "${option}"
    in
        h)
            usage
            exit 0
            ;;
        d) FLOTTA_DASHBOARD=${OPTARG};;
        u)
            echo "Uninstalling Grafana"
            uninstall_grafana
            exit 0
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

if [[ -z $FLOTTA_DASHBOARD ]]; then
    FLOTTA_DASHBOARD="./docs/metrics/flotta-dashboard.json"
    echo "No dashboard specified, using default: $FLOTTA_DASHBOARD"
fi

if [ ! -f "$FLOTTA_DASHBOARD" ]; then
  echo "File $FLOTTA_DASHBOARD does not exist"
  exit 1
fi

GRAFANA_OPERATOR=grafana-operator.v4.2.0

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
  startingCSV: ${GRAFANA_OPERATOR}
EOF

kubectl wait subscription -n flotta grafana-operator --for condition=CatalogSourcesUnhealthy=False --timeout=60s
echo "Waiting for Grafana operator to be ready"
while [ "$(kubectl get csv -n flotta ${GRAFANA_OPERATOR} -o jsonpath='{.status.phase}')" != "Succeeded" ]; do
    echo -n "."
    sleep 5
done
echo $'\n'"Grafana operator is ready"
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
     admin_password: admin
     admin_user: admin
 ingress:
   enabled: true
EOF

echo "Waiting for Grafana instance to be ready"
while [ "$(kubectl get grafana.integreatly.org/grafana -n flotta -o jsonpath='{.status.message}')" != "success" ]
do
    echo -n "."
    sleep 5
done
echo $'\n'"Grafana instance is ready"
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

GRAFANA_URL=$(kubectl get routes -n flotta grafana-route --no-headers -o=custom-columns=HOST:.spec.host)
GRAFANA_API="https://admin:admin@${GRAFANA_URL}/api"
echo "Waiting for Grafana server to be ready at $GRAFANA_API"
count=0
until [[ count -gt 20 ]]
do
  curl -k -s -i "$GRAFANA_API/search" | grep "200 OK" > /dev/null
  if [ "$?" == "1" ]; then
    echo -n "."
    count=$((count+1))
    sleep 5
  else
    echo $'\n'"Grafana server is ready"
    break
  fi
done

# the purpose of this sleep is to let grafana handler be ready. without it, the creation of the dashboard is reported as
# successful but nothing is actually created
sleep 20

$(dirname $0)/import_grafana_dashboards.sh -d $FLOTTA_DASHBOARD -g $GRAFANA_API

echo -e "Grafana dashboard URL:\033[92m https://$GRAFANA_URL\033[0m"