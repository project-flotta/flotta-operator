# Custom Resource Definitions

## EdgeDevice

`EdgeDevice` is a namespaced custom resource that represents registered edge device and its configuration.

* apiVersion: `management.k4e.io/v1alpha1`
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
```

### Status

```yaml
status:
  dataObc: 242e48d0-286b-4170-9b97-95502066e6ae # Name of the Object Bucket Claim created for this device
  lastSeenTime: "2021-09-23T09:27:50Z" # Time of tha last heartbeat message
  lastSyncedResourceVersion: "13040122" # Version of configuration applied on the device as reported in the latest heartbeat message 
  phase: up # phase of edge device's lifecycle
  deployments: # list of workloads deployed to the device
    - name: nginx # name of the workload (corresponds to EdgeDeployment CR in the same namespace)
      phase: Running # workload status (Deploying, Running, Created, etc.);
      lastTransitionTime: "2021-09-23T09:27:50Z" # last time when state of the workload changed  
      lastDataUpload: "2021-09-23T09:27:30Z" # time of the latest successful data upload for the workload 
      
  hardware: # Hardware configuration information; CPU, memory, GPU, network interfaces, disks, etc.
    ...

```
For more information about the `dataObc` property read about the [Data Upload](data-upload.md) feature.

## EdgeDeployment

`EdgeDeployment` is a namespaced custom resource that represents workload that should be deployed to edge devices matching criteria specified in the CR.

* apiVersion: `management.k4e.io/v1alpha1`
* kind: `EdgeDeployment`

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
  type: pod # type of the deployment; currently only pod is supported
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