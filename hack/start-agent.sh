#!/bin/bash

usage()
{
cat << EOF
Usage: $0 OPTIONS
This script runs test for flotta-agent to verify it can start running properly.
OPTIONS:
   -d      Path to flotta-device-worker directory (optional)
   -h      Show this message
   -t      Target (ocp/k8s)
   -y      Path to yggdrasil directory
EOF
}

while getopts "t:y:d:" option; do
    case "${option}"
    in
        t) TARGET=${OPTARG};;
        y) YGGDRASIL_PATH=${OPTARG};;
        d) DEVICE_WORKER_PATH=${OPTARG};;
        h)
            usage
            exit 0
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

if [[ -z $TARGET ]]; then
    echo "Error: TARGET is required"
    exit 1
fi

if [[ -z $YGGDRASIL_PATH ]]; then
    echo "Error: YGGDRASIL_PATH is required"
    exit 1
fi

if [[ -n $DEVICE_WORKER_PATH ]]; then
  cd $DEVICE_WORKER_PATH
fi

sudo systemctl enable --now nftables.service
sudo systemctl enable --now podman.service
sudo systemctl enable --now podman.socket

make uninstall
make install

cd $YGGDRASIL_PATH
export PREFIX=/var/local
make bin

pid_file="${PREFIX}/var/run/yggdrasil/workers/device-worker.pid"
if [ -f "$pid_file" ] ; then
    sudo rm "$pid_file"
fi

if [[ "$TARGET" == "k8s" ]]; then
  sudo ./yggd \
    --protocol http \
    --path-prefix api/flotta-management/v1 \
    --client-id $(cat /etc/machine-id) \
    --cert-file /tmp/cert.pem \
    --key-file /tmp/key.pem \
    --ca-root /tmp/ca.pem \
    --log-level trace \
    --server project-flotta.io:443

elif [[ "$TARGET" == "ocp" ]]; then
  HTTP_SERVER=$(oc get routes flotta-operator-controller-manager -n flotta --no-headers -o=custom-columns=HOST:.spec.host)
  sudo ./yggd \
    --log-level info \
    --protocol http \
    --path-prefix api/flotta-management/v1 \
    --client-id $(cat /etc/machine-id) \
    --cert-file /tmp/cert.pem \
    --key-file /tmp/key.pem \
    --server $HTTP_SERVER
fi

echo start-agent done!
