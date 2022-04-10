# Device metrics

Edge devices can collect metrics from all deployed workloads and the system of the device itself. The collected metrics are sent to a `metrics receiver`. This document describes how that functionality can
be configured.

## System metrics

### Collection frequency
System metrics collection is enabled by default and the Flotta agent will start gathering them when the device is started - 
with default intervals of **60** seconds. Said interval can be customized by setting desired frequency (in seconds) in
an `EdgeDevice` CR.

For instance, following spec snippet would instruct the device worker to collect system metrics every 5 minutes.

```yaml
spec:
  metrics:
    system:
      interval: 300
```

### Allow-lists
By default, the device worker would collect only pre-defined, narrow list of system metrics; user can modify the set of collected metrics using *system metrics allow-list*.

Allow-list configuration comprises two elements: 
 - `ConfigMap` containing a list of metrics to be collected (exclusively)
 - Reference to the above `ConfigMap` in the `EdgeDevice` system metrics configuration

Sample allow-list `ConfigMap` (mind `metrics_list.yaml` key):

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: system-allow-list
  namespace: devices
data:
  metrics_list.yaml: |
    names: 
      - node_disk_io_now
      - node_memory_Mapped_bytes
      - node_network_speed_bytes
```

Reference to the above `ConfigMap` in an `EdgeDevice` spec:

```yaml
spec:
  metrics:
    system:
      allowList: 
          name: system-allow-list
```
## Metrics receiver
### Overview
The devices can be configured to write the metrics to a remote server. The client in the device uses [Prometheus Remote Write API](https://docs.google.com/document/d/1LPhVRSFkGNSuU1fBd81ulhsCPR4hkSZyyBj1SZ8fWOM/edit#heading=h.p12mxouu8g0h) (see also [Prometheus Integrations](https://prometheus.io/docs/operating/integrations/)). The device writes metrics until it reaches the end of the TSDB contents. It then waits 5 minutes for more metrics to be collected.
### Configuration
The feature is disabled by default. It is configured via EdgeDevice/EdgeDeviceGroup CRs. Example with inline documentation and defaults:
```yaml
spec:
    metrics:
      receiverConfiguration:
        caSecretName: receiver-tls # secret containing CA cert. Secret key is 'ca.crt'. Optional
        requestNumSamples: 10000 # maximum number of samples in each request from device to receiver. Optional
        timeoutSeconds: 10 # timeout for requests to receiver. Optional
        url: https://receiver:19291/api/v1/receive # the receiver's URL. Used to indicate HTTP/HTTPS. Set to empty in order to disable writing to receiver
```
### Example receiver
We prepared an example for deploying a [Thanos](https://thanos.io) receiver.
Example includes deployment with and without TLS.
The receiver listens on port `19291` for incoming writes.
The deployment's pod includes a container that executes a Thanos querier. You can use it for querying the received metrics. It listens on port `9090`.

Without TLS:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: thanos-receiver
  labels:
    app: thanos-receiver
spec:
  replicas: 1
  selector:
    matchLabels:
      app: thanos-receiver
  template:
    metadata:
      labels:
        app: thanos-receiver
    spec:
      containers:
      - name: receive
        image: quay.io/thanos/thanos:v0.24.0
        command:
        - /bin/thanos
        - receive
        - --label
        - "receiver=\"0\""
      - name: query
        image: quay.io/thanos/thanos:v0.24.0
        command:
        - /bin/thanos
        - query
        - --http-address
        - 0.0.0.0:9090
        - --grpc-address
        - 0.0.0.0:11901
        - --endpoint
        - 127.0.0.1:10901
```
With TLS:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: thanos-receiver
  labels:
    app: thanos-receiver
spec:
  replicas: 1
  selector:
    matchLabels:
      app: thanos-receiver
  template:
    metadata:
      labels:
        app: thanos-receiver
    spec:
      initContainers:
      - name: http-config
        image: fedora
        command: ["/bin/sh"]
        args: ["-c", "echo -e \"tls_server_config:\\n  cert_file: /etc/server-tls/tls.crt\\n  key_file: /etc/server-tls/tls.key\" > /etc/shared/http.config"]
        volumeMounts:
        - name: shared
          mountPath: /etc/shared
      containers:
      - name: receive
        image: quay.io/thanos/thanos:v0.24.0
        command:
        - /bin/thanos
        - receive
        - --label
        - "receiver=\"0\""
        - --remote-write.server-tls-cert
        - /etc/server-tls/tls.crt
        - --remote-write.server-tls-key
        - /etc/server-tls/tls.key
        volumeMounts:
        - name: server-tls
          mountPath: /etc/server-tls
      - name: query
        image: quay.io/thanos/thanos:v0.24.0
        command:
        - /bin/thanos
        - query
        - --http-address
        - 0.0.0.0:9090
        - --grpc-address
        - 0.0.0.0:11901
        - --endpoint
        - 127.0.0.1:10901
        - --http.config
        - /etc/shared/http.config
        volumeMounts:
        - name: server-tls
          mountPath: /etc/server-tls
        - name: shared
          mountPath: /etc/shared
      volumes:
      - name: server-tls
        secret:
          secretName: thanos-receiver-tls
      - name: shared
        emptyDir: {}
```