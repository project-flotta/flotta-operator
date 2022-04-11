#!/bin/bash

usage()
{
cat << EOF
Usage: $0 OPTIONS
This script runs sanity check for project-flotta for testing flotta-operator is up and running properly.
OPTIONS:
   -h      Show this message
   -i      Image registry
   -o      Path to flotta-operator directory (optional)
   -t      Target (ocp/k8s)
EOF
}

while getopts "i:t:o:" option; do
    case "${option}"
    in
        i) IMG=${OPTARG};;
        t) TARGET=${OPTARG};;
        o) OPERATOR_PATH=${OPTARG};;
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

if [[ -z $IMG ]]; then
    echo "Error: IMG is required"
    usage
    exit 1
fi

if [[ -z $TARGET ]]; then
    echo "Error: TARGET is required"
    usage
    exit 1
fi

if [[ -n $OPERATOR_PATH ]]; then
  cd $OPERATOR_PATH
fi

sudo rm /tmp/*.pem
make get-certs
sudo chown root:root /tmp/*.pem
IMG=$IMG make docker-build docker-push
make install-router
IMG=$IMG TARGET=$TARGET make deploy

echo start-operator done!
