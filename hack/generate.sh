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
    rm -rf client models
    "${CONTAINER_RUNTIME}" run "${CONTAINER_RUNTIME_OPTS}" -u $(id -u):$(id -u) -v ${__root}:${__root}:rw,Z -v /etc/passwd:/etc/passwd -w ${__root} \
        quay.io/goswagger/swagger:v0.25.0 generate client -f swagger.yaml --template=stratoscale
}

function generate_go_server() {
    rm -rf restapi
    "${CONTAINER_RUNTIME}" run "${CONTAINER_RUNTIME_OPTS}" -u $(id -u):$(id -u) -v ${__root}:${__root}:rw,Z -v /etc/passwd:/etc/passwd -w ${__root} \
        quay.io/goswagger/swagger:v0.25.0 generate server  -f ${__root}/swagger.yaml --template=stratoscale
}

function generate_docs() {
    "${CONTAINER_RUNTIME}" run "${CONTAINER_RUNTIME_OPTS}" -u $(id -u):$(id -u) -v ${__root}:${__root}:rw,Z -v /etc/passwd:/etc/passwd -w ${__root} \
        quay.io/goswagger/swagger:v0.27.0 generate markdown  -f ${__root}/swagger.yaml --template=stratoscale --output=docs/design/http-api-swagger.md
}

function generate_from_swagger() {
    generate_go_client
    generate_go_server
    generate_docs
}

"$@"
