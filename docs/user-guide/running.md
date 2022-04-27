# Running Flotta

The components need to be started in following order:
 - Operator
 - Agent
   - Install flotta-device-worker
   - Run yggdrasil

When yggdrasil with flotta-device-worker installed is started, it registers the device in the Operator -  EdgeDevice CR representing the device is automatically created in the k8s cluster.

When your environment is running, you can [deploy your workload](deploying-workloads.md).

---
**Warning** 

To force device re-registration remove EdgeDevice CR from the cluster and `/var/local/yggdrasil/device` directory from the device

---

## Manually

### Operator

#### Prerequisites

 - k8s cluster with OpenShift Cluster Storage (or at least NooBaa) to run;
 - Checkout https://github.com/project-flotta/flotta-operator
 - Make sure that your `oc` or `kubectl` is configured properly to communicate with your cluster

#### Installation

1. Build and push operator images:
   
   Run in the flotta-operator repository
   
   `IMG=<image repository and tag> make docker-build docker-push`

    for example:

   `IMG=quay.io/jdzon/flotta-operator:latest make docker-build docker-push`
2. Deploy the operator:

   Run in the flotta-operator repository
 
   `IMG=<image repository and tag> make install deploy`
3. Forward flotta-operator ports to allow the agent to communicate with it:
 
   `kubectl port-forward service/flotta-operator-controller-manager -n flotta 8043 --address 0.0.0.0`

4. Dump the registration cert using the makefile target:

```
make get-certs
```

5. For running with Yggdrasil, the certs should have the correct permissions
```
sudo chown root:root /tmp/*.pem
```

### Agent from RPMs

#### Prepare installation scripts

On a machine with KUBECONFIG pointing at cluster with Flotta Operator running, execute in the `flotta-operator` repository:
```shell
make agent-install-scripts
```

As a result, two files will be generated:
 - `hack/install-agent-rpm-ostree.sh` - to be used to install Flotta Agent on machines with Ostree-based system (i.e. Fedora IoT);
 - `hack/install-agent-dnf.sh` - to be used to install Flotta Agent on machines with `dnf` package manager installed (i.e. Fedora Server); 

#### Upload chosen file

Upload your installation script to machine that should be managed by Flotta.

#### Install

Execute uploaded script, passing as an argument IP address of machine exposing Flotta HTTP API port 8043:

```shell
./install-agent-rpm-ostree.sh -i <HTTP_API_IP>
```

The script will install and enable required software and machine will get registered as an `EdgeDevice` as soon as agent 
components are up.

### Agent - from sources

#### yggdrasil

##### Prerequisites

- Checkout https://github.com/RedHatInsights/yggdrasil repository on main branch

##### Running

---
**Warning**

Following step should be done after flotta-device-worker is installed on the edge device

---

Start yggdrasil with from the yggdrasil repository directory:

`
sudo yggd \
  --protocol http \
  --path-prefix api/flotta-management/v1 \
  --client-id $(cat /etc/machine-id) \
  --cert-file /tmp/cert.pem \
  --key-file /tmp/key.pem \
  --ca-root /tmp/ca.pem \
  --server 127.0.0.1:8043
`

When running Yggdrasil, a new edgedevice should be added into the list

```
kubectl get edgedevices
```

And the certificate should be renewed, with one that it's specific for that
device.

```
make check-certs
```


#### flotta-device-worker

##### Prerequisites

On the build machine:

- Install following packages:
  - go 
  - btrfs-progs-devel
  - gpgme-devel
  - device-mapper-devel
  - podman
- Checkout https://github.com/project-flotta/flotta-device-worker repository 

On the edge device:
 
- Make sure Podman is running
  ```shell
  systemctl --now enable --user podman.socket
  loginctl enable-linger $(USER)
  ```

  Verify with:
  ```shell
  systemctl status --user podman.socket
  ```
- Make sure `/usr/local/libexec/yggdrasil` directory exists


##### Building and installing on the edge device

1. `export GOPROXY=proxy.golang.org,direct`
2. `LIBEXECDIR=/usr/local/libexec make install`

##### Alternative: Building on a separate build machine

###### Building
Execute following steps on the build machine:

1. `export GOPROXY=proxy.golang.org,direct`
2. `make build`

###### Installing 

1. Upload <flotta-device-worker repo dir>/bin/device-worker to `/usr/local/libexec/yggdrasil` directory on the edge device
2. Make sure that `/usr/local/libexec/yggdrasil/device-worker` is executable


## Using edge device ISO

1. Follow [ISO generation scripts](https://github.com/ydayagi/r4e) documentation to create installation ISO containing the agent
2. Boot your device with generated ISO
3. The system is installed automatically
4. The agent installed on the device will register it with control plane configured in the first step
