# Design

## Architecture

![](architecture.png)

### Control plane - Kubernetes

#### Operator

The Flotta operator is a Kubernetes operator that consists of two components:
 - controller responsible for reconciling `EdgeDevice` and `EdgeDeployment` CRs;
 - HTTP API that is used by the Flotta agent to get expected configuration and to post heartbeat messages. See [HTTP API schema](http-api.md) for more details.

#### Object Storage

Object Storage is used to store files created by workloads on devices and uploaded using Flotta built-in mechanism.

Object Bucket Claims provided by OCS and Noobaa and accessible with like S3 buckets. 

See [Data Upload](data-upload.md) for more details.

#### CRDs

Operator manages two kinds of CRs: 
 - `EdgeDevice` representing physical edge devices; see [the definition](../../config/crd/bases/management.project-flotta.io_edgedevices.yaml) or [an example](../../config/samples/management_v1alpha1_edgedevice.yaml)
 - `EdgeDeployment` representing workloads that can be scheduled to run on edge devices; see [the definition](../../config/crd/bases/management.project-flotta.io_edgedeployments.yaml) or [an example](../../config/samples/management_v1alpha1_edgedeployment.yaml)

See [Custom Resource Definitions](crds.md) for more details.


### Edge Devices

#### Agent

Agent is running constantly on the edge device and responsible for:
 - getting expected device configuration specification from the control plane;
 - making the device's actual state to reflect configuration specified by the control plane;
   - re-configuring agent internal settings;
   - starting, removing and monitoring workloads - pods; 
 - posting device's status to control plane using heartbeat messages;
 - uploading workload-generated files to control plane Object Storage (see [Data Upload](data-upload.md)).

Agent is running in two processes:
 - [yggdrasil](https://github.com/jakub-dzon/yggdrasil/) - gateway service responsible for communication with the control plane; dispatches messages to and from the control plane from/to workers (extensions);
 - [device-worker](https://github.com/project-flotta/flotta-device-worker) - yggdrasil worker responsible for performing all the logic listed above.


#### Communication

```mermaid
%%{init: {'theme':'base'}}%%
sequenceDiagram
autonumber
participant Agent as Device Agent
participant YGGD as Yggdrasil
participant API as Operator API

rect rgb(226, 232, 238)
  Agent ->> YGGD: Register (GRPC+Unixsocket)
  YGGD ->> Agent: 200 OK
  Agent ->> YGGD: Send with registration directive
  YGGD ->> API: Data message with registration directive
  API ->> YGGD: 200 OK with signed certificate
  YGGD ->> Agent: 200 OK with signed certificate.
  Note over YGGD, Agent: New client certificate writen
end

rect rgb(233, 237, 202)
  loop Heartbeat
    Agent -->> YGGD: Heartbeat
    YGGD-->> API: Heartbeat
    API -->> YGGD: 200 OK
    YGGD-->> Agent: 200 OK
  end
end
```

1. Flotta device-agent(Agent) register to Yggdrasil using Unixsocket and GRPC
   connection
2. Yggdrasil allows device-worker to register in the platform.
3. Agent needs to register to the API, and for that sends a data-message with
   the registration directive that includes a Certificate Signed Request(CSR).
4. Yggdrasil forward that message in the configured transport to the Operator API
5. If registration is successful, it will return a signed certificate based on
   the CSR
6. Yggdrasil sends to the agent the response. Agent rewrite the client certs to
   create new connections with the device certificates.
7. -10 Each minute, Agent sends heartbeat data to the operator, if the operator
returns 200 OK nothing happens, if returns 401 for the certificate expired the
registration process will be started again.

- **What a registration certificate can do?**

Just register, at any other operation will fail and it's only capable of doing
registration with the registration directive. The same happens with the expired
device certificate, that can only re-register again to issue a new certificate.


## Workflows

### Device registration/pairing

```mermaid
sequenceDiagram
autonumber
actor User
participant Device
participant Agent
participant Operator

User ->> Device: Boot Flotta device ISO with registration certificates
Device ->> Agent: Start Yggdrasil agent
Agent ->> Operator: Send registration request
Operator ->> "EdgeDevice CR": Create
Operator ->> Agent: Pairing confirmation with new certificates signed
Operator ->> OBC: Create Object Bucket Claim for the device data
Operator ->> "EdgeDevice CR": Update status with OBC info
Note over Device,Operator: New HTTPS MTLS connection
Agent ->> Agent: Start processing requests

loop Daemon loop each minute
Agent ->> Operator: Sending Heartbeats
Operator ->> Agent: 200 OK
end
```
![](pairing.png)

 1. User boots the edge device with Flotta device ISO that includes registration
    certificates.
 2. Agent service is started by systemd
 3. Agent sends pairing/registration request containing device's hardware information and the certificate signed request for their certificates to the control plane (Operator's HTTP endpoint) using registration certificates.
 4. Operator creates `EdgeDevice` resource representing the registering device
 5. Agent registration is concluded and device switch to the new certificate.
 6. Operator creates `ObjectBucketClaim` for storing data uploaded from the device
 7. Operator updates `EdgeDevice` status sub-resource with the name of newly created `ObjectBucketClaim`
 8. Agent starts processing requests (downloading configuration)
 9. Agent schedules workload data directories monitoring
 10. Agent schedules periodical heartbeat messages

### Worklad deployment workflow

![](workload_deployment.png)

 1. User creates `WorkloadDeployment` resource
 2. Operator process `WorkloadDeployment` resource and finds matching `EdgeDevices`
 3. Operator adds `WorkloadDeployment` reference to all matching `EdgeDevices`
 4. New workload configuration is included in configuration downloaded by the agent
 5. Deployment status in `EdgeDevice` status sub-resource is set to "Deploying"
 6. Agent opens host ports listed in the pod specification
 7. Agent starts the pod on the device using podman play kube
 8. New workload status is included in periodically sent heartbeat messages
 9. The workload status in `EdgeDevice` status sub-resource is updated
