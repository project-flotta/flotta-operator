# HTTP API

Operator's HTTP API consists of four endpoints used in communication with yggdrasil/agent:

 - `GET /data/{device_id}/in`
 - `POST /data/{device_id}/out`
 - `GET /control/{device_id}/in`
 - `POST /control/{device_id}/out`

## `GET /data/{device_id}/in`

This endpoint is used by the agent to retrieve its expected configuration; the response is `message` object described in the [Swagger specification](http_api_swagger.md). 
The `content` field of the message contains payload understood by the device-worker. The payload is described by the `device-configuration-message` Swagger object. 

The `content` is forwarded to the `device-worker` and processed there.


## `POST /data/{device_id}/out` 

This endpoint is used by the agent to send information to the operator. Currently, there are only two types of message contents supported by this endpoint (see [Swagger specification](http_api_swagger.md)):

 - `registration-info` - sent by the device once, when it registers with the cluster
 - `heartbeat` - sent periodically to report device and its workloads status to the cluster

## `GET /control/{device_id}/in`

This endpoint is used by the agent to retrieve "control" commands. 

Supported commands:
 - `disconnect` - device upon reception of that command MUST stop communicating with the control plane and remove all the workloads and related artifacts;

## `POST /control/{device_id}/out`

This endpoint is used by the agent to send "control" commands. Currently, all commands are ignored by the operator.