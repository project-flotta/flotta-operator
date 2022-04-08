# Custom Resource Definitions

## EdgeDevice

`EdgeDevice` is a namespaced custom resource that represents registered edge device and its configuration.

* apiVersion: `management.project-flotta.io/v1alpha1`
* kind: `EdgeDevice`

### Specification

```yaml
spec:
  heartbeat:
    periodSeconds: 5 # Interval in seconds with which the heartbeat messages should be sent from the agent 
    hardwareProfile: # Defines the scope of hardware information sent with the heartbeat messages; currently unused
      include: true # Specifies whether the hardware should be sent at all
      scope: full # Specifies how much information should be provided; "full" - everything; "delta" - only changes compared to the previous updated
  requestTime: "2021-09-22T08:35:25Z" # Time of the device registration request
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

### Status

```yaml
status:
  dataObc: 242e48d0-286b-4170-9b97-95502066e6ae # Name of the Object Bucket Claim created for this device
  lastSeenTime: "2021-09-23T09:27:50Z" # Time of tha last heartbeat message
  lastSyncedResourceVersion: "13040122" # Version of configuration applied on the device as reported in the latest heartbeat message 
  phase: up # phase of edge device's lifecycle
  workloads: # list of workloads deployed to the device
    - name: nginx # name of the workload (corresponds to EdgeWorkload CR in the same namespace)
      phase: Running # workload status (Deploying, Running, Created, etc.);
      lastTransitionTime: "2021-09-23T09:27:50Z" # last time when state of the workload changed  
      lastDataUpload: "2021-09-23T09:27:30Z" # time of the latest successful data upload for the workload 
      
  hardware: # Hardware configuration information; CPU, memory, GPU, network interfaces, disks, etc.
    ...

```
For more information about the `dataObc` property read about the [Data Upload](data-upload.md) feature.

## EdgeWorkload

`EdgeWorkload` is a namespaced custom resource that represents workload that should be deployed to edge devices matching criteria specified in the CR.

* apiVersion: `management.project-flotta.io/v1alpha1`
* kind: `EdgeWorkload`

### Specification

```yaml
spec:
  deviceSelector: # Specifies which EdgeDevice CRs this workload should be deployed to. See https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#resources-that-support-set-based-requirements
    matchLabels: # Mutually exclusive with matchExpressions
      dc: home
    matchExpressions: # Mutually exclusive with matchLabels
      - key: dc
        operator: In
        value: [home]
  type: pod # type of the workload; currently only pod is supported
  data: # See below for details
    paths:
      - source: stats # well-known "/export" container directory sub-path (/export/stats in this case) that should be periodically uploaded to the control plane   
        target: statistics # path of the directory in control plane storage where the data should be uploaded to (currently - statistics directory in edge device's OBC) 
  pod:
    spec: # Pod configuration as described in https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/
      containers:
        - name: random-server
          image: quay.io/jdzon/random-server:v1
          ports:
            - containerPort: 8080
              hostPort: 9090
```

#### Data Upload
Go to this document to read about the [Data Upload](data-upload.md) feature.

#### Pod specification caveats

* `containers[].envFrom` - env variables referencing is not supported
* `containers[].ports.hostPort` - has to be specified to be opened on the host and being forwarded to the `containerPort`
* only `volumes[].hostPath` and `volumes[].persistentVolumeClaim` volume types are supported
* `volumes[].hostPath.CharDevice` and `volumes[].hostPath.BlockDevice` `hostPath` volume subtypes are not supported
* **TBD**

## EdgeDeviceSet

`EdgeDeviceSet` is a namespaced custom resource that represents edge device configuration that can be assigned to multiple devices at the same time and allows for centralized configuration management.

* apiVersion: `management.project-flotta.io/v1alpha1`
* kind: `EdgeDeviceSet`

### Specification

`EdgeDeviceSet` specification is a copy of selected parts of `EdgeDevice` specification.

```yaml
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

## EdgeConfig
`EdgeConfig` is a namespaced custom resource that represents custom configuration that should be deployed to edge devices matching criteria specified in the CR.

* apiVersion: `management.project-flotta.io/v1alpha1`
* kind: `EdgeConfig`

```yaml
spec:
  deviceIDs: # The deviceID list on which the playbook should be executed. Necessary to execute playbook on a devices that don't belong to any group
    - device-1
    - device-2
  edgePlaybook:
    ansiblePlaybookCmd:
      user: foo # Username who execute the playbook
      playbooksPriorityMap:
        - "1" : 
          content: # Ansible playbook in base64
          LS0tCi0gIG5hbWU6IEhlbGxvIFdvcmxkIEFuc2libGUgUGxheWJvb2sKICAgaG9zdHM6IDEyNy4wLjAu....
          timeoutSeconds: 100 # Interval in seconds on which the playbook execution should be executed
          requiredPrivilegeLevel: # The required privelege level necessary to execute the playbook. See https://man7.org/linux/man-pages/man7/capabilities.7.html for a full list
            capAdd: # Capabilities to add
              - SYS_CHROOT
            capDrop: # Capabilities to drop
              - SYS_BOOT
          ansibleOptions:
            check: false #  Flag defining whether the playbook execution should be in check mode (is just a simulation)
          privilegeEscalationOptions: # To execute tasks with root privileges or with another user’s permissions
            become: true # Flag definig whether to activate privilege escalation
            becomeUser: bar # set to user with desired privileges
            becomeMethod: sudo # method to use to increase privilege. Currently only `sudo` and `su` are available
          executionStategy: ExecuteOnce # Define the execution strategy for the playbook. Currently `StopOnFailure`, `RetryOnFailure`, `ExecuteOnce` are available.
        - "2" : 
          content: # Ansible playbook in base64 
          LS0tCgotIGhvc3RzOiBhbGwKICBnYXRoZXJfZmFjdHM6IGZhbHNlCiAgdmFyczoKICAgIGFycmF5OgogICAgICAtFlvdXIg....
          ...
```

### Status


```yaml
status:
  condition:
    - 
      type: TargetVerification
      status: false
      reason: # one-word CamelCase reason for the condition's last transition
      message: # human-readable message indicating details about last transition
        Verifying playbook for rhel4edge target environment
      lastTransitionTime: "2021-09-23T09:27:50Z" # last time the condition transit from one status to another 
    - 
      type: Deploying
      status: false
      lastTransitionTime:
    - 
      type: Executing
      status: false
      lastTransitionTime:
    - 
      type: Completed
      status: false
      reason:
      message: "Execution completed with error. Execution strategy: RetryOnFailure"
      lastTransitionTime:
    -
      type: Completed
      status: true
      reason: "Execution completed with error. Execution strategy: ExecuteOnce"
      message:
      lastTransitionTime:
```