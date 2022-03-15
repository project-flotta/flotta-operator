# Podman auto-update

Auto update containers according to their auto-update policy.

Auto-update looks up containers with a specified `io.containers.autoupdate` label. This label is set by the `flotta-operator`.
Flotta-operator looks up labels on `EdgeDeployments` CR prefixed by `podman/` string. If such label is found it's propageted to
the podman pod and containers.

If  the  label  is  present  and set to registry, Podman reaches out to the corresponding registry to check if the image has been updated. The label image is an alternative to registry maintained for
backwards compatibility.  An image is considered updated if the digest in the local storage is different than the one of the remote image.  If an image must be  updated,  Podman  pulls  it  down  and
restarts the systemd unit executing the container.

If `ImageRegistries.AuthFileSecret` is defined, Podman reaches out to the corresponding authfile when pulling images.

Edgedevice start the `podman-auto-update.timer` which is responsible for executing the `podman-auto-update.service`. This unit is triggered daily.

To create a Edgedeployment with auto-update feature enabled define following label:
```yaml
apiVersion: management.project-flotta.io/v1alpha1
kind: EdgeDeployment
metadata:
  name: nginx
  labels:
    podman/io.containers.autoupdate: registry
spec:
  deviceSelector:
    matchLabels:
      device: local
  type: pod
  pod:
    spec:
      containers:
        - name: nginx
          image: quay.io/bitnami/nginx:1.20
```
