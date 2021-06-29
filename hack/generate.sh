#!/usr/bin/env bash

set -o nounset
set -o pipefail
set -o errexit

set -x

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__root="$(cd "$(dirname "${__dir}")" && pwd)"

function generate_go_client() {
    rm -rf client models
    docker run -u $(id -u):$(id -u) -v ${__root}:${__root}:rw,Z -v /etc/passwd:/etc/passwd -w ${__root} \
        quay.io/goswagger/swagger:v0.25.0 generate client -f swagger.yaml --template=stratoscale
}

function generate_go_server() {
    rm -rf restapi
    docker run -u $(id -u):$(id -u) -v ${__root}:${__root}:rw,Z -v /etc/passwd:/etc/passwd -w ${__root} \
        quay.io/goswagger/swagger:v0.25.0 generate server  -f ${__root}/swagger.yaml --template=stratoscale
}

function generate_from_swagger() {
    generate_go_client
    generate_go_server
}

"$@"