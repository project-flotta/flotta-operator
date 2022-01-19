# Device metrics

Edge devices can collect metrics from all deployed workloads and the system of the device itself. This document describes how that functionality can
be configured.

## System metrics

### Collection frequency
System metrics collection is enabled by default and the k4e agent will start gathering them when the device is started - 
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
          namespace: devices
```