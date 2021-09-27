# Installation
 To install the operator [CRDs](docs/crds.md) in the cluster pointed to by kubectl configuration execute (it may take some time):

`make install`

When the CRDs are present in the cluster, the operator can either be run outside or inside the cluster.

# Running
## Outside the cluster
Run `make run` to start the operator.

## Inside the cluster
### Deployment
 - Build and push operator images:
   
   `IMG=<image repository and tag> make docker-build docker-push` 
   
   for example: `IMG=quay.io/jdzon/k4e-operator:latest make docker-build docker-push`
   
 - Deploy the operator:
   
   `IMG=<image repository and tag> make deploy`

### Port forwarding
The port used by the operator HTTP API (`8888`) has to be available outside the cluster, so following port-forwarding command needs to be executed before attempting to communicate with it:

`kubectl port-forward service/k4e-operator-controller-manager -n k4e-operator-system 8888`
