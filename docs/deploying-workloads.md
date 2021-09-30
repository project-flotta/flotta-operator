# Deploying workloads on registered edge devices

Workload can be deployed to devices __after__ they are registered with the cluster.

`EdgeDeployment` can be deployed only on edge devices in the same namespace.


## Deploying by device ID

---
**Note**

When both methods are used in one `EdgeDeployment`, this one takes precedence over the selector-based one.

---

`EdgeDeployment` can be deployed on one chosen device by specifying its name in `spec.device` property.

For example, for `EdgeDevice` `242e48d0-286b-4170-9b97-95502066e6ae`, following property should be set in `EdgeDeployment` yaml:

```yaml
spec:
  ...
  device: 242e48d0-286b-4170-9b97-95502066e6ae
  ...
```

## Deploying with selector

`EdgeDeployment` can be installed on multiple devices using label selectors.

To install your workload using this method:

1. Label chosen `EdgeDevice` objects;

   For example:

   `oc label edgedevice 242e48d0-286b-4170-9b97-95502066e6ae dc=emea`

2. Select `dc=emea` label in the `EdgeDeployment` specification:

   ```yaml
   spec:
     deviceSelector:
       matchLabels:
         dc: emea
       
   ```
   
   or
   
   ```yaml
   spec:
     deviceSelector:
       matchExpressions:
         - key: dc
           operator: In
           values: [emea]
       
   ```
   
   The second approach can be used for matching multiple values of one label. For example:
   
   ```yaml
   spec:
     deviceSelector:
       matchExpressions:
         - key: dc
           operator: In
           values: [emea, apac]
       
   ```

3. Create the `EdgeDeployment` in the cluster:
   `kubectl apply -f your-deployment.yaml`

## Inspecting workload status

To check statuses of all workloads deployed to an edge device:

```shell
oc get edgedevice <edge device name> -ojsonpath="{range .status.deployments[*]}{.name}{':\t'}{.phase}{'\n'}{end}"
```

To list all devices having chosen workload deployed:

```shell
oc get edgedevice -l workload/<workload-name>="true"
```

`EdgeDevice` is labeled with `workload/<workload-name>="true"` when `EdgeDeployment` is added to it.