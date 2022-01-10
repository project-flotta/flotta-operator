# Installation
 To install the operator [CRDs](docs/design/crds.md) in the cluster pointed to by kubectl configuration execute (it may take some time):

`make install`

When the CRDs are present in the cluster, the operator can either be run outside or inside the cluster.

# Getting started
## Outside the cluster
Run `make run` to start the operator.

## Inside the cluster
### Deployment
 - Build and push operator images:
   
   `IMG=<image repository and tag> make docker-build docker-push` 
   
   for example: `IMG=quay.io/jdzon/k4e-operator:latest make docker-build docker-push`
 
### On OpenShift cluster
- Deploy the operator: `IMG=<image registry and tag> TARGET=ocp make deploy`.
- Get HTTP server address by running: `HTTP_SERVER=$(oc get routes k4e-operator-controller-manager -n k4e-operator-system --no-headers -o=custom-columns=HOST:.spec.host)`.
- Start yggdrasil with from the yggdrasil repository directory: `sudo go run ./cmd/yggd --log-level info --transport http --cert-file /etc/pki/consumer/cert.pem --key-file /etc/pki/consumer/key.pem --client-id-source machine-id --http-server $HTTP_SERVER`.

### On minikube
- Deploy the operator: `IMG=<image registry and tag> HOST=<host name> TARGET=k8s make deploy`.
- Add to /etc/hosts: `<minikube ip> <host name>`.
- Start yggdrasil with from the yggdrasil repository directory: `sudo go run ./cmd/yggd --log-level info --transport http --cert-file /etc/pki/consumer/cert.pem --key-file /etc/pki/consumer/key.pem --client-id-source machine-id --http-server <host name>`.

In order to change the verbosity of the logger check out [here](docs/user-guide/logger.md). 

For additional resources check out: [metrics](docs/metrics/metrics.md), [grafana dashboard](docs/metrics/grafana.md).
