# Running k4e

The components need to be started in following order:
 - Operator
 - Agent
   - Install k4e-device-worker
   - Run yggdrasil

When yggdrasil with k4e-device-worker installed is started, it registers the device in the Operator -  EdgeDevice CR representing the device is automatically created in the k8s cluster.

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
 
   `kubectl port-forward service/flotta-operator-controller-manager -n flotta 8888 --address 0.0.0.0`

### yggdrasil

#### Prerequisites

- Generate certificate and a key:

  `openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out cert.pem -keyout key.pem`
- Put generated cert.pem and key.pem files in `/etc/pki/consumer` edge device directory 
- Checkout https://github.com/jakub-dzon/yggdrasil repository

#### Running

---
**Warning**

Following step should be done after k4e-device-worker is installed on the edge device

---

Start yggdrasil with from the yggdrasil repository directory:

`sudo go run ./cmd/yggd --log-level trace --transport http --cert-file /etc/pki/consumer/cert.pem --key-file /etc/pki/consumer/key.pem --client-id-source machine-id --http-server <your.k8s-ingress:8888>`

### k4e-device-worker

#### Prerequisites

On the build machine:

- Install following packages:
  - go 
  - btrfs-progs-devel
  - gpgme-devel
  - device-mapper-devel
  - podman
- Checkout https://github.com/jakub-dzon/k4e-device-worker repository 

On the edge device:
 
- Make sure Podman is running
  ```shell
  sudo systemctl --now enable podman.socket
  sudo loginctl enable-linger root
  ```

  Verify with:
  ```shell
  systemctl status podman.socket
  ```
- Make sure `/usr/local/libexec/yggdrasil` directory exists


#### Building and installing on the edge device

1. `export GOPROXY=proxy.golang.org,direct`
2. `LIBEXECDIR=/usr/local/libexec make install`

#### Alternative: Building on a separate build machine

##### Building
Execute following steps on the build machine:

1. `export GOPROXY=proxy.golang.org,direct`
2. `make build`

##### Installing 

1. Upload <k4e-device-worker repo dir>/bin/device-worker to `/usr/local/libexec/yggdrasil` directory on the edge device
2. Make sure that `/usr/local/libexec/yggdrasil/device-worker` is executable




## Using edge device ISO

1. Follow [ISO generation scripts](https://github.com/ydayagi/r4e) documentation to create installation ISO containing the agent
2. Boot your device with generated ISO
3. The system is installed automatically
4. The agent installed on the device will register it with control plane configured in the first step