# Profiling Flotta Operator

In order to enable profiling for all Flotta Operator components by pprof, few steps are required:

Add the following to the start of the __main.go__ file:
```go
mux := http.NewServeMux()
mux.HandleFunc("/debug/pprof/", pprof.Index)
mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

go func() {
    http.ListenAndServe(":6060", mux)
}()
```
Add required import as well to the __main.go__ file:
```go
import _ "net/http/pprof"
```

The profiling data is exposed on port 6060 of the Flotta Operator pod, once built and deployed.

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