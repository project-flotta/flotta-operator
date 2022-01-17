


# Kube4EdgeManagement
Kube for Edge Management
  

## Informations

### Version

1.0.0

### License

[Apache 2.0](http://www.apache.org/licenses/LICENSE-2.0.html)

## Tags

  ### <span id="tag-devices"></span>devices

Device management

## Content negotiation

### URI Schemes
  * http
  * https

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  yggdrasil

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| GET | /api/k4e-management/v1/control/{device_id}/in | [get control message for device](#get-control-message-for-device) |  |
| GET | /api/k4e-management/v1/data/{device_id}/in | [get data message for device](#get-data-message-for-device) |  |
| POST | /api/k4e-management/v1/control/{device_id}/out | [post control message for device](#post-control-message-for-device) |  |
| POST | /api/k4e-management/v1/data/{device_id}/out | [post data message for device](#post-data-message-for-device) |  |
  


## Paths

### <span id="get-control-message-for-device"></span> get control message for device (*GetControlMessageForDevice*)

```
GET /api/k4e-management/v1/control/{device_id}/in
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| device_id | `path` | string | `string` |  | ✓ |  | Device ID |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-control-message-for-device-200) | OK | Success |  | [schema](#get-control-message-for-device-200-schema) |
| [401](#get-control-message-for-device-401) | Unauthorized | Unauthorized |  | [schema](#get-control-message-for-device-401-schema) |
| [403](#get-control-message-for-device-403) | Forbidden | Forbidden |  | [schema](#get-control-message-for-device-403-schema) |
| [404](#get-control-message-for-device-404) | Not Found | Error |  | [schema](#get-control-message-for-device-404-schema) |
| [500](#get-control-message-for-device-500) | Internal Server Error | Error |  | [schema](#get-control-message-for-device-500-schema) |

#### Responses


##### <span id="get-control-message-for-device-200"></span> 200 - Success
Status: OK

###### <span id="get-control-message-for-device-200-schema"></span> Schema
   
  

[Message](#message)

##### <span id="get-control-message-for-device-401"></span> 401 - Unauthorized
Status: Unauthorized

###### <span id="get-control-message-for-device-401-schema"></span> Schema

##### <span id="get-control-message-for-device-403"></span> 403 - Forbidden
Status: Forbidden

###### <span id="get-control-message-for-device-403-schema"></span> Schema

##### <span id="get-control-message-for-device-404"></span> 404 - Error
Status: Not Found

###### <span id="get-control-message-for-device-404-schema"></span> Schema

##### <span id="get-control-message-for-device-500"></span> 500 - Error
Status: Internal Server Error

###### <span id="get-control-message-for-device-500-schema"></span> Schema

### <span id="get-data-message-for-device"></span> get data message for device (*GetDataMessageForDevice*)

```
GET /api/k4e-management/v1/data/{device_id}/in
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| device_id | `path` | string | `string` |  | ✓ |  | Device ID |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-data-message-for-device-200) | OK | Success |  | [schema](#get-data-message-for-device-200-schema) |
| [401](#get-data-message-for-device-401) | Unauthorized | Unauthorized |  | [schema](#get-data-message-for-device-401-schema) |
| [403](#get-data-message-for-device-403) | Forbidden | Forbidden |  | [schema](#get-data-message-for-device-403-schema) |
| [404](#get-data-message-for-device-404) | Not Found | Error |  | [schema](#get-data-message-for-device-404-schema) |
| [500](#get-data-message-for-device-500) | Internal Server Error | Error |  | [schema](#get-data-message-for-device-500-schema) |

#### Responses


##### <span id="get-data-message-for-device-200"></span> 200 - Success
Status: OK

###### <span id="get-data-message-for-device-200-schema"></span> Schema
   
  

[Message](#message)

##### <span id="get-data-message-for-device-401"></span> 401 - Unauthorized
Status: Unauthorized

###### <span id="get-data-message-for-device-401-schema"></span> Schema

##### <span id="get-data-message-for-device-403"></span> 403 - Forbidden
Status: Forbidden

###### <span id="get-data-message-for-device-403-schema"></span> Schema

##### <span id="get-data-message-for-device-404"></span> 404 - Error
Status: Not Found

###### <span id="get-data-message-for-device-404-schema"></span> Schema

##### <span id="get-data-message-for-device-500"></span> 500 - Error
Status: Internal Server Error

###### <span id="get-data-message-for-device-500-schema"></span> Schema

### <span id="post-control-message-for-device"></span> post control message for device (*PostControlMessageForDevice*)

```
POST /api/k4e-management/v1/control/{device_id}/out
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| device_id | `path` | string | `string` |  | ✓ |  | Device ID |
| message | `body` | [Message](#message) | `models.Message` | | ✓ | |  |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-control-message-for-device-200) | OK | Success |  | [schema](#post-control-message-for-device-200-schema) |
| [401](#post-control-message-for-device-401) | Unauthorized | Unauthorized |  | [schema](#post-control-message-for-device-401-schema) |
| [403](#post-control-message-for-device-403) | Forbidden | Forbidden |  | [schema](#post-control-message-for-device-403-schema) |
| [404](#post-control-message-for-device-404) | Not Found | Error |  | [schema](#post-control-message-for-device-404-schema) |
| [500](#post-control-message-for-device-500) | Internal Server Error | Error |  | [schema](#post-control-message-for-device-500-schema) |

#### Responses


##### <span id="post-control-message-for-device-200"></span> 200 - Success
Status: OK

###### <span id="post-control-message-for-device-200-schema"></span> Schema

##### <span id="post-control-message-for-device-401"></span> 401 - Unauthorized
Status: Unauthorized

###### <span id="post-control-message-for-device-401-schema"></span> Schema

##### <span id="post-control-message-for-device-403"></span> 403 - Forbidden
Status: Forbidden

###### <span id="post-control-message-for-device-403-schema"></span> Schema

##### <span id="post-control-message-for-device-404"></span> 404 - Error
Status: Not Found

###### <span id="post-control-message-for-device-404-schema"></span> Schema

##### <span id="post-control-message-for-device-500"></span> 500 - Error
Status: Internal Server Error

###### <span id="post-control-message-for-device-500-schema"></span> Schema

### <span id="post-data-message-for-device"></span> post data message for device (*PostDataMessageForDevice*)

```
POST /api/k4e-management/v1/data/{device_id}/out
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| device_id | `path` | string | `string` |  | ✓ |  | Device ID |
| message | `body` | [Message](#message) | `models.Message` | | ✓ | |  |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-data-message-for-device-200) | OK | Success |  | [schema](#post-data-message-for-device-200-schema) |
| [400](#post-data-message-for-device-400) | Bad Request | Error |  | [schema](#post-data-message-for-device-400-schema) |
| [401](#post-data-message-for-device-401) | Unauthorized | Unauthorized |  | [schema](#post-data-message-for-device-401-schema) |
| [403](#post-data-message-for-device-403) | Forbidden | Forbidden |  | [schema](#post-data-message-for-device-403-schema) |
| [404](#post-data-message-for-device-404) | Not Found | Error |  | [schema](#post-data-message-for-device-404-schema) |
| [500](#post-data-message-for-device-500) | Internal Server Error | Error |  | [schema](#post-data-message-for-device-500-schema) |

#### Responses


##### <span id="post-data-message-for-device-200"></span> 200 - Success
Status: OK

###### <span id="post-data-message-for-device-200-schema"></span> Schema

##### <span id="post-data-message-for-device-400"></span> 400 - Error
Status: Bad Request

###### <span id="post-data-message-for-device-400-schema"></span> Schema

##### <span id="post-data-message-for-device-401"></span> 401 - Unauthorized
Status: Unauthorized

###### <span id="post-data-message-for-device-401-schema"></span> Schema

##### <span id="post-data-message-for-device-403"></span> 403 - Forbidden
Status: Forbidden

###### <span id="post-data-message-for-device-403-schema"></span> Schema

##### <span id="post-data-message-for-device-404"></span> 404 - Error
Status: Not Found

###### <span id="post-data-message-for-device-404-schema"></span> Schema

##### <span id="post-data-message-for-device-500"></span> 500 - Error
Status: Internal Server Error

###### <span id="post-data-message-for-device-500-schema"></span> Schema

## Models

### <span id="boot"></span> boot


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| current_boot_mode | string| `string` |  | |  |  |
| pxe_interface | string| `string` |  | |  |  |



### <span id="container-metrics"></span> container-metrics


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| disabled | boolean| `bool` |  | |  |  |
| path | string| `string` |  | | Path to use when retrieving metrics |  |
| port | int32 (formatted integer)| `int32` |  | | Port to use when retrieve the metrics |  |



### <span id="cpu"></span> cpu


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| architecture | string| `string` |  | |  |  |
| count | integer| `int64` |  | |  |  |
| flags | []string| `[]string` |  | |  |  |
| frequency | number| `float64` |  | |  |  |
| model_name | string| `string` |  | |  |  |



### <span id="data-configuration"></span> data-configuration


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| paths | [][DataPath](#data-path)| `[]*DataPath` |  | |  |  |



### <span id="data-path"></span> data-path


> Device-to-control plane paths mapping
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| source | string| `string` |  | | Path in the workload container |  |
| target | string| `string` |  | | Path in the control plane storage |  |



### <span id="device-configuration"></span> device-configuration


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| heartbeat | [HeartbeatConfiguration](#heartbeat-configuration)| `HeartbeatConfiguration` |  | |  |  |
| metrics | [MetricsConfiguration](#metrics-configuration)| `MetricsConfiguration` |  | |  |  |
| storage | [StorageConfiguration](#storage-configuration)| `StorageConfiguration` |  | |  |  |



### <span id="device-configuration-message"></span> device-configuration-message


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| configuration | [DeviceConfiguration](#device-configuration)| `DeviceConfiguration` |  | |  |  |
| device_id | string| `string` |  | | Device identifier |  |
| secrets | [SecretList](#secret-list)| `SecretList` |  | | List of secrets used by the workloads |  |
| version | string| `string` |  | |  |  |
| workloads | [WorkloadList](#workload-list)| `WorkloadList` |  | | List of workloads deployed to the device |  |
| workloads_monitoring_interval | integer| `int64` |  | | Defines the interval in seconds between the attempts to evaluate the workloads status and restart those that failed |  |



### <span id="disk"></span> disk


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| bootable | boolean| `bool` |  | |  |  |
| by_id | string| `string` |  | | by-id is the World Wide Number of the device which guaranteed to be unique for every storage device |  |
| by_path | string| `string` |  | | by-path is the shortest physical path to the device |  |
| drive_type | string| `string` |  | |  |  |
| hctl | string| `string` |  | |  |  |
| id | string| `string` |  | | Determine the disk's unique identifier which is the by-id field if it exists and fallback to the by-path field otherwise |  |
| io_perf | [IoPerf](#io-perf)| `IoPerf` |  | |  |  |
| is_installation_media | boolean| `bool` |  | | Whether the disk appears to be an installation media or not |  |
| model | string| `string` |  | |  |  |
| name | string| `string` |  | |  |  |
| path | string| `string` |  | |  |  |
| serial | string| `string` |  | |  |  |
| size_bytes | integer| `int64` |  | |  |  |
| smart | string| `string` |  | |  |  |
| vendor | string| `string` |  | |  |  |
| wwn | string| `string` |  | |  |  |



### <span id="event-info"></span> event-info


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| message | string| `string` |  | | Message describe the event which has occured. |  |
| reason | string| `string` |  | | Reason is single word description of the subject of the event. |  |
| type | string| `string` |  | | Either 'info' or 'warn', which reflect the importance of event. |  |



### <span id="gpu"></span> gpu


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| address | string| `string` |  | | Device address (for example "0000:00:02.0") |  |
| device_id | string| `string` |  | | ID of the device (for example "3ea0") |  |
| name | string| `string` |  | | Product name of the device (for example "UHD Graphics 620 (Whiskey Lake)") |  |
| vendor | string| `string` |  | | The name of the device vendor (for example "Intel Corporation") |  |
| vendor_id | string| `string` |  | | ID of the vendor (for example "8086") |  |



### <span id="hardware-info"></span> hardware-info


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| boot | [Boot](#boot)| `Boot` |  | |  |  |
| cpu | [CPU](#cpu)| `CPU` |  | |  |  |
| disks | [][Disk](#disk)| `[]*Disk` |  | |  |  |
| gpus | [][Gpu](#gpu)| `[]*Gpu` |  | |  |  |
| hostname | string| `string` |  | |  |  |
| interfaces | [][Interface](#interface)| `[]*Interface` |  | |  |  |
| memory | [Memory](#memory)| `Memory` |  | |  |  |
| system_vendor | [SystemVendor](#system-vendor)| `SystemVendor` |  | |  |  |



### <span id="hardware-profile-configuration"></span> hardware-profile-configuration


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| include | boolean| `bool` |  | |  |  |
| scope | string| `string` |  | |  |  |



### <span id="heartbeat"></span> heartbeat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| events | [][EventInfo](#event-info)| `[]*EventInfo` |  | | Events produced by device worker. |  |
| hardware | [HardwareInfo](#hardware-info)| `HardwareInfo` |  | | Hardware information |  |
| status | string| `string` |  | |  |  |
| time | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| version | string| `string` |  | |  |  |
| workloads | [][WorkloadStatus](#workload-status)| `[]*WorkloadStatus` |  | |  |  |



### <span id="heartbeat-configuration"></span> heartbeat-configuration


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| hardware_profile | [HardwareProfileConfiguration](#hardware-profile-configuration)| `HardwareProfileConfiguration` |  | |  |  |
| period_seconds | integer| `int64` |  | |  |  |



### <span id="image-registries"></span> image-registries


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| authFile | string| `string` |  | | Image registries authfile created by executing `podman login` or `docker login` (i.e. ~/.docker/config.json). https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#log-in-to-docker-hub describes how the file can be created and how it is structured. |  |



### <span id="interface"></span> interface


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| biosdevname | string| `string` |  | |  |  |
| client_id | string| `string` |  | |  |  |
| flags | []string| `[]string` |  | |  |  |
| has_carrier | boolean| `bool` |  | |  |  |
| ipv4_addresses | []string| `[]string` |  | |  |  |
| ipv6_addresses | []string| `[]string` |  | |  |  |
| mac_address | string| `string` |  | |  |  |
| mtu | integer| `int64` |  | |  |  |
| name | string| `string` |  | |  |  |
| product | string| `string` |  | |  |  |
| speed_mbps | integer| `int64` |  | |  |  |
| vendor | string| `string` |  | |  |  |



### <span id="io-perf"></span> io_perf


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| sync_duration | integer| `int64` |  | | 99th percentile of fsync duration in milliseconds |  |



### <span id="memory"></span> memory


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| physical_bytes | integer| `int64` |  | |  |  |
| usable_bytes | integer| `int64` |  | |  |  |



### <span id="message"></span> message


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| content | [interface{}](#interface)| `interface{}` |  | | Content |  |
| directive | string| `string` |  | |  |  |
| message_id | string| `string` |  | |  |  |
| metadata | [interface{}](#interface)| `interface{}` |  | |  |  |
| response_to | string| `string` |  | |  |  |
| sent | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| type | string| `string` |  | |  |  |
| version | integer| `int64` |  | |  |  |



### <span id="metrics"></span> metrics


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| containers | map of [ContainerMetrics](#container-metrics)| `map[string]ContainerMetrics` |  | |  |  |
| interval | integer| `int64` |  | | Interval(in seconds) to scrape metrics endpoint. |  |
| path | string| `string` |  | | Path to use when retrieving metrics |  |
| port | int32 (formatted integer)| `int32` |  | |  |  |



### <span id="metrics-configuration"></span> metrics-configuration


> Defines metrics configuration for the device
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| retention | [MetricsRetention](#metrics-retention)| `MetricsRetention` |  | | Defines metrics data retention limits |  |
| system | [SystemMetricsConfiguration](#system-metrics-configuration)| `SystemMetricsConfiguration` |  | | System metrics gathering configuration |  |



### <span id="metrics-retention"></span> metrics-retention


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| max_hours | int32 (formatted integer)| `int32` |  | | Maximum time in hours metrics data files should kept on the device |  |
| max_mib | int32 (formatted integer)| `int32` |  | | Maximum size of metrics stored on disk |  |



### <span id="registration-info"></span> registration-info


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| hardware | [HardwareInfo](#hardware-info)| `HardwareInfo` |  | | Hardware information |  |
| os_image_id | string| `string` |  | |  |  |



### <span id="s3-storage-configuration"></span> s3-storage-configuration


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| aws_access_key_id | string| `string` |  | |  |  |
| aws_ca_bundle | string| `string` |  | |  |  |
| aws_secret_access_key | string| `string` |  | |  |  |
| bucket_host | string| `string` |  | |  |  |
| bucket_name | string| `string` |  | |  |  |
| bucket_port | int32 (formatted integer)| `int32` |  | |  |  |
| bucket_region | string| `string` |  | |  |  |



### <span id="secret"></span> secret


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| data | string| `string` |  | | The secret's data section in JSON format |  |
| name | string| `string` |  | | Name of the secret |  |



### <span id="secret-list"></span> secret-list


  

[][Secret](#secret)

### <span id="storage-configuration"></span> storage-configuration


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| s3 | [S3StorageConfiguration](#s3-storage-configuration)| `S3StorageConfiguration` |  | |  |  |



### <span id="system-metrics-configuration"></span> system-metrics-configuration


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| interval | int32 (formatted integer)| `int32` |  | | Interval(in seconds) to scrape metrics endpoint. |  |



### <span id="system-vendor"></span> system_vendor


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| manufacturer | string| `string` |  | |  |  |
| product_name | string| `string` |  | |  |  |
| serial_number | string| `string` |  | |  |  |
| virtual | boolean| `bool` |  | | Whether the machine appears to be a virtual machine or not |  |



### <span id="workload"></span> workload


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| data | [DataConfiguration](#data-configuration)| `DataConfiguration` |  | | Configuration for data transfer |  |
| imageRegistries | [ImageRegistries](#image-registries)| `ImageRegistries` |  | | Image registries configuration |  |
| metrics | [Metrics](#metrics)| `Metrics` |  | | Metrics endpoint configuration |  |
| name | string| `string` |  | | Name of the workload |  |
| specification | string| `string` |  | |  |  |



### <span id="workload-list"></span> workload-list


  

[][Workload](#workload)

### <span id="workload-status"></span> workload-status


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| last_data_upload | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| name | string| `string` |  | |  |  |
| status | string| `string` |  | |  |  |


