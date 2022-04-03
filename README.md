# Installation
 To install the operator [CRDs](docs/design/crds.md) in the cluster pointed to by kubectl configuration execute (it may take some time):

`make install`

When the CRDs are present in the cluster, the operator can either be run outside or inside the cluster.

## Prerequisites/Dependencies
### Go
The operator requires Go `1.16`, Go `1.18` is not supported yet.

### podman
This project relies strongly on Podman. It is used in the edge devices and also in the operator.
Supported Podman version is `3.4.4`

### cert-manager
We use admission webhooks for managing the CRDs. Therefore, there's a need for TLS certificates and keys management.
When deploying in a cluster, webhooks are enabled and require [cert-manager](https://cert-manager.io/). Refer to its installation instructions for installing it.

Disabling webhooks is done by setting environment variable `ENABLE_WEBHOOKS=false` for the manager's container.

# Getting started
## Outside the cluster
Run `make run` to start the operator.

## Inside the cluster
### Deployment
 - Build and push operator images:
   
   `IMG=<image repository and tag> make docker-build docker-push` 
   
   for example: `IMG=quay.io/jdzon/flotta-operator:latest make docker-build docker-push`
 
### On OpenShift cluster
- Deploy the operator: `IMG=<image registry and tag> TARGET=ocp make deploy`.
- Get HTTP server address by running: `HTTP_SERVER=$(oc get routes flotta-operator-controller-manager -n flotta --no-headers -o=custom-columns=HOST:.spec.host)`.
- Start yggdrasil with from the yggdrasil repository directory: 
   ```
    sudo ./yggd \
    --log-level info \
    --protocol http \
    --path-prefix api/flotta-management/v1 \
    --client-id $(cat /etc/machine-id) \
    --cert-file /etc/pki/consumer/cert.pem \
    --key-file /etc/pki/consumer/key.pem \
    --server $HTTP_SERVER
   ```

### On minikube
- Deploy the operator: `IMG=<image registry and tag> HOST=<host name> TARGET=k8s make deploy`.
- Add to /etc/hosts: `<minikube ip> <host name>`.
- Start yggdrasil with from the yggdrasil repository directory:
   ```
    sudo ./yggd \
    --log-level info \
    --protocol http \
    --path-prefix api/flotta-management/v1 \
    --client-id $(cat /etc/machine-id) \
    --cert-file /etc/pki/consumer/cert.pem \
    --key-file /etc/pki/consumer/key.pem \
    --server $HOST
   ```

In order to change the verbosity of the logger check out [here](docs/user-guide/logger.md).

For additional resources check out: [metrics](docs/metrics/metrics.md), [grafana dashboard](docs/metrics/grafana.md), [device metrics](docs/user-guide/device-metrics.md), [image registries](docs/user-guide/image-registries-auth.md).
