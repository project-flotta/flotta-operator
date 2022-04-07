# Device grouping

Edge Devices present in the cluster can be grouped using `EdgeDeviceGroup` CR. That CR would then provide common 
configuration for all the edge devices using it and **override any configuration in scope that is present on the 
`EdgeDevice` CR**.

Currently `EdgeDeviceGroup` CRs do not take part in workload scheduling and no changes made to `EdgeDeviceGroup` 
or group membership impact the way workloads are deployed. 

## EdgeDeviceGroup configuration scope

User may define following configuration elements using `EdgeDeviceGroup`:

 - heartbeat (`spec.heartbeat`);
 - metrics (`spec.metrics`);
 - data transfer (`spec.storage`);
 - log collection (`spec.logCollection`);
 - OS configuration (`spec.osInformation`).

Full `EdgeDeviceGroup` CR might look as follows:

```yaml
apiVersion: management.project-flotta.io/v1alpha1
kind: EdgeDeviceGroup
metadata:
  name: sample-group
spec:
  heartbeat:
    periodSeconds: 5 # Interval in seconds with which the heartbeat messages should be sent from the agent 
    hardwareProfile: # Defines the scope of hardware information sent with the heartbeat messages; currently unused
      include: true # Specifies whether the hardware should be sent at all
      scope: full # Specifies how much information should be provided; "full" - everything; "delta" - only changes compared to the previous updated
  metrics:
    retention:
      maxMiB: 200 # Specifies how much disk space should be used for storing persisted metrics on the device
      maxHours: 24 # Specifies how long should persisted metrics be stored on the device disk
    system:
      interval: 60 # Interval in seconds with which the device system metrics should be collected
      allowList:
        name: allow-list-map # Defines name of a ConfigMap containing list of system metrics that should be scraped
        disabled: true #  When set to true instructs the device to turn off system metrics collection
  osInformation:
    automaticallyUpgrade: true # Flag defining whether OS upgrade should be performed automatically when the commitID changes
    commitID: 0305686e69d673cb15ad459990fab4a3e4c5aba1 # Commit ID of desired OS ostree version for the device
    hostedObjectsURL: http://images.project-flotta.io # URL of the hosted commits web server
  storage:
    s3:
      secretName: common-s3-secret # Name of the secret containing S3 API access credentials
      configMapName: common-s3-config #Name of a config map containing S3 API access configuration options
      createOBC: false # Flag defining whether the OBC should be automatically created for the device (if this feature is disabled for the operator)
  logCollection:
    syslog:
      kind: syslog # Kind of a log collection system. Currently, only `syslog` is available
      bufferSize: 12 # Size of a log sending buffer in kilobytes
      syslogConfig: 
        name: syslog-config-map # Name of a config map containing syslog connection configuration
```

## Defining `EdgeDevice` - `EdgeDeviceGroup` relationship

To make specific `EdgeDevice` use chosen `EdgeDeviceGroup` configuration, user needs to add 
`flotta/configured-by: <edge device group name>` label to the `EdgeDevice`.
For example, if there is a `group-1` `EdgeDeviceGroup` and `device-1` `EdgeDevice`, user needs to issue following 
command to build the relationship between them:
```shell
kubectl label edgedevice device-1 flotta/configured-by=group-1
```

## Configuration priority

Configuration defined in the `EdgeDeviceGroup` **always** takes precedence over whatever is defined in the `EdgeDevice` 
as a whole. It means that even if some element is not present in the `EdgeDeviceGroup`, usual default values are used for it, 
even if it is defined at the level of `EdgeDevice`.

Changes made to the `EdgeDeviceGroup` will be applied to all edge devices using it.

If corresponding `EdgeDeviceGroup` cannot be found, `EdgeDevice` configuration is used.

## Example

### `EdgeDevice` spec
```yaml
spec:
  heartbeat:
    hardwareProfile:
      include: true
    periodSeconds: 60
```

### `EdgeDeviceGroup` spec
```yaml
spec:
  heartbeat:
    periodSeconds: 15
  metrics:
    system:
      interval: 600
  osInformation:
    automaticallyUpgrade: true
```

### Effective device configuration

Spec used to generate device configuration:
```yaml
spec:
  heartbeat:
    periodSeconds: 15
  metrics:
    system:
      interval: 600
  osInformation:
    automaticallyUpgrade: true
```

Configuration sent to the device (with default applied):
```json
"configuration": {
  "heartbeat": {
   "hardware_profile": {},
   "period_seconds": 15
  },
  "metrics": {
   "receiver": {
    "request_num_samples": 30000,
    "timeout_seconds": 10
   },
   "system": {
    "interval": 600
   } 
  },
  "os": {
   "automatically_upgrade": true
  }
 }
```