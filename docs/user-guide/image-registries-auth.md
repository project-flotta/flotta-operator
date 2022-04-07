# Container Image Registries Credentials

Container images used by `EdgeWorkloads` may be hosted in private, protected repositories requiring clients to present 
correct credentials. Users can provide said credentials to Flotta operator as a `Secret` referred to by an `EdgeWorkload`.

Secret should contain correct docker auth file under the `.dockerconfigjson` key:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pull-secret
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: ewoJImF1dGhzIjogewoJCSJxdWF5LmlvIjogewoJCQkiYXV0aCI6ICJabTl2SUdKaGNpQm1iMjhnWW1GeUNnPT0iCgkJfQoJfQp9Cg==
```

where `.dockerconfigjson` is a base64 encoded Docker auth file JSON:

```json
{
  "auths": {
    "quay.io": {
      "auth": "Zm9vIGJhciBmb28gYmFyCg=="
    }
  }
}
```

The above JSON can be taken from file created by running `podman login`, `docker login` or similar.

Secret created above must be placed in the same namespace as `EdgeWorkload` using it. 

Reference to the above `Secret` in an `EdgeWorkload` spec: 

```yaml
spec:
  imageRegistries:
    secretRef:
      name: pull-secret
```