# Running Flotta integration tests locally

#### Prerequisites

 - k8s cluster with OpenShift Cluster Storage (or at least NooBaa) to run;
 - Checkout https://github.com/project-flotta/flotta-operator
 - Make sure that your `oc` or `kubectl` is configured properly to communicate with your cluster

#### Running the tests

1. Create container image of flotta operator.

```bash
# Switch to flotta git repository
$ cd $FLOTTA_OPERATOR_GIT_REPO

# Build the flotta operator
$ make build

# Make sure you have configured k8s provider docker
# For example for minikube by running: eval $(minikube -p minikube docker-env)
# Then run the build of the container
$ IMG=flotta-operator:latest make docker-build

# Deploy the operator on k8s
$ make deploy IMG=flotta-operator

# Wait until the operator is ready
$ kubectl wait --timeout=120s --for=condition=Ready pods --all -n flotta
```

2. Expose the flotta API

```bash
$ kubectl port-forward deploy/flotta-operator-controller-manager -n flotta --address 0.0.0.0 8888:8888 &
```

3. Run the tests

```bash
$ make integration-test
```

#### Troubleshooting

1. kubectl wait timeouts:

   If timeout, debug the deployment logs by running:

   `kubectl logs deploy/flotta-operator-controller-manager -n flotta`

2. Waiting for edge device timeouts in integration tests:

   Debug the edge device container by executing shell inside the container:

   `docker exec -it edgedevice1 /bin/bash`
   
   Then run any debugging command like:
   `journalctl -u yggdrasild.service`
