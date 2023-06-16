#!/usr/bin/env bash

set -o pipefail
set -o errexit

set -x

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__root="$(cd "$(dirname "${__dir}")" && pwd)"

if which podman 2>/dev/null >&2; then
    export CONTAINER_RUNTIME="podman"
    export CONTAINER_RUNTIME_OPTS="--userns=keep-id"
elif which docker 2>/dev/null >&2; then
    export CONTAINER_RUNTIME="docker"
else
    echo No container runtime found, install podman or docker.
    exit 2
fi

function generate_go_client() {
    rm -rf client models backend/client
    mkdir -p backend/client
    "${CONTAINER_RUNTIME}" run -u $(id -u):$(id -u) -e GOCACHE=/tmp -v ${__root}:${__root}:rw,Z  -w ${__root} \
        quay.io/goswagger/swagger:v0.29.0 generate client -f ${__root}/swagger.yaml --template=stratoscale
    "${CONTAINER_RUNTIME}" run -u $(id -u):$(id -u) -e GOCACHE=/tmp -v ${__root}:${__root}:rw,Z  -w ${__root} \
        quay.io/goswagger/swagger:v0.29.0 generate client -t backend -f ${__root}/swagger-backend.yaml --template=stratoscale
}

function generate_go_server() {
    rm -rf restapi backend/restapi
    mkdir -p backend/restapi
    "${CONTAINER_RUNTIME}" run -u $(id -u):$(id -u) -e GOCACHE=/tmp -v ${__root}:${__root}:rw,Z  -w ${__root} \
        quay.io/goswagger/swagger:v0.29.0 generate server  -f ${__root}/swagger.yaml --template=stratoscale
    "${CONTAINER_RUNTIME}" run -u $(id -u):$(id -u) -e GOCACHE=/tmp -v ${__root}:${__root}:rw,Z  -w ${__root} \
        quay.io/goswagger/swagger:v0.29.0 generate server  -t backend -f ${__root}/swagger-backend.yaml --template=stratoscale
}

function generate_from_swagger() {
    generate_go_client
    generate_go_server
}

"$@"
