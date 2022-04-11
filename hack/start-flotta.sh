#!/bin/bash

usage()
{
cat << EOF
Usage: $0 OPTIONS
This script runs test for project-flotta to verify all flotta components can start running properly.
OPTIONS:
   -d      Path to flotta-device-worker directory
   -h      Show this message
   -i      Image registry
   -o      Path to flotta-operator directory
   -t      Target (ocp\k8s)
   -y      Path to yggdrasil directory
EOF
}

while getopts "i:t:o:d:y:" option; do
    case "${option}"
    in
        i) IMG=${OPTARG};;
        t) TARGET=${OPTARG};;
        o) OPERATOR_PATH=${OPTARG};;
        d) DEVICE_WORKER_PATH=${OPTARG};;
        y) YGGDRASIL_PATH=${OPTARG};;
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

if [[ -z $OPERATOR_PATH ]]; then
    echo "Error: OPERATOR_PATH is required"
    exit 1
fi

if [[ -z $DEVICE_WORKER_PATH ]]; then
    echo "Error: DEVICE_WORKER_PATH is required"
    exit 1
fi

if [[ -z $YGGDRASIL_PATH ]]; then
    echo "Error: YGGDRASIL_PATH is required"
    exit 1
fi

if [[ -z $IMG ]]; then
    echo "Error: IMG is required"
    exit 1
fi

if [[ -z $TARGET ]]; then
    echo "Error: TARGET is required"
    exit 1
fi

sh hack/start-operator.sh -i $IMG -t $TARGET -o $OPERATOR_PATH
if [ $? -eq 1 ]; then
    echo "Error: start operator"
else
    sh hack/start-agent.sh -t $TARGET -y $YGGDRASIL_PATH -d $DEVICE_WORKER_PATH
    if [ $? -eq 1 ]; then
        echo "Error: start agent"
    else
        echo "Done!"
    fi
fi
