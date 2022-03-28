#!/bin/sh

usage()
{
cat << EOF
usage: $0 options
This script will import given Grafana dashboard to Grafana.
OPTIONS:
   -h      Show this message
   -d      The dashboard to import
   -g      Grafana API URL
EOF
}

while getopts "hd:g:" option; do
    case "${option}"
    in
        h)
            usage
            exit 0
            ;;
        d) FLOTTA_DASHBOARD=${OPTARG};;
        g) GRAFANA_API=${OPTARG};;
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

if [[ -z $GRAFANA_API ]]; then
  GRAFANA_URL=$(kubectl get routes -n flotta grafana-route --no-headers -o=custom-columns=HOST:.spec.host)
  GRAFANA_API="https://admin:admin@${GRAFANA_URL}/api"
fi
echo "Using Grafana server: $GRAFANA_API"

request_body=$(mktemp)
cat <<EOF >> $request_body
{
  "dashboard": $(cat $FLOTTA_DASHBOARD),
  "overwrite": true
}
EOF
curl -s -X POST --insecure -H "Content-Type: application/json" -d @$request_body "$GRAFANA_API/dashboards/import"

echo $'\n'"Grafana dashboard imported"