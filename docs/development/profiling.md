# Profiling Flotta Operator

By default, Flotta Operator disables profiling. To enable profiling, set the `ENABLE_PROFILING` environment variable to `true`:

```bash
kubectl patch cm -n flotta flotta-operator-manager-config --type merge --patch '{"data":{"ENABLE_PROFILING": "true"}}'
```

This will enable profiling for all Flotta Operator components by pprof.
The profiling data is exposed on port 6060 of the Flotta Operator pod.
To view the profiling data, and visualize it nicely, use [`pyroscope`](https://pyroscope.io/).

To install `pyroscope`:
```bash
helm repo add pyroscope-io https://pyroscope-io.github.io/helm-chart
helm install pyroscope pyroscope-io/pyroscope -f https://raw.githubusercontent.com/pyroscope-io/pyroscope/main/examples/golang-pull/kubernetes/values.yaml
kubectl expose svc pyroscope

# for openshift, get the address for pyroscope service:
kubectl get route pyroscope kubectl get routes pyroscope --no-headers -o=custom-columns=HOST:.spec.host 
```

There is need to patch flotta's resources to expose targets and mark pods for profiling:
```bash
kubectl patch deployment flotta-operator-controller-manager -n flotta -p '
  { "spec": {
      "template": {
        "spec":
          { "containers":
            [{"name": "manager",
              "ports": [
                  {
                      "containerPort": 6060,
                      "name": "pprof",
                      "protocol": "TCP"
                  }
              ]
            }]
          }
        }
      }
  }'

  kubectl patch service flotta-operator-controller-manager -n flotta -p '
  { "spec": {
      "ports": [
          {
              "name": "pprof",
              "port": 6060,
              "protocol": "TCP",
              "targetPort": "pprof"
          }
      ]
  }
  }'

  kubectl patch deployment -n flotta flotta-operator-controller-manager -p '
   {
     "spec": {
       "template":{
         "metadata":{
           "annotations":{
             "pyroscope.io/scrape": "true",
             "pyroscope.io/application-name": "flotta-operator",
             "pyroscope.io/profile-cpu-enabled": "true",
             "pyroscope.io/profile-mem-enabled": "true",
             "pyroscope.io/port": "6060"
           }
         }
       }
     }
  }'
```

Restart the Flotta Operator to apply the changes.