# Installation
 To install the operator [CRDs](docs/crds.md) in the cluster pointed to by kubectl configuration execute (it may take some time):

`make install`

When the CRDs are present in the cluster, the operator can either be run outside or inside the cluster.

# Running
## Outside the cluster
Run `make run` to start the operator.

## Inside the cluster
### Deployment
 - Build and push operator images:
   
   `IMG=<image repository and tag> make docker-build docker-push` 
   
   for example: `IMG=quay.io/jdzon/k4e-operator:latest make docker-build docker-push`
   
 - Deploy the operator:
   
   `IMG=<image repository and tag> make deploy`

### Port forwarding
The port used by the operator HTTP API (`8888`) has to be available outside the cluster, so following port-forwarding command needs to be executed before attempting to communicate with it:

`kubectl port-forward service/k4e-operator-controller-manager -n k4e-operator-system 8888`

## Change the verbosity of the logger

To change the verbosity of the logger, the user can update the value of the `LOG_LEVEL` field.
Admitted values are: 	`debug`, `info`, `warn`, `error`, `dpanic`, `panic`, and `fatal`.
Refer to [zapcore docs](https://github.com/uber-go/zap/blob/v1.15.0/zapcore/level.go#L32) for details on each log level.

For example:\
`kubectl patch cm -n k4e-operator-system k4e-operator-manager-config --type merge --patch '{"data":{"LOG_LEVEL": "debug"}}'`

In case of:
-  _Inside the cluster_ run, the pod will be automatically restarted.\
-  _Outside the cluster_ run, the user must set the `LOG_LEVEL` field and manually restart the operator.

### Implementation details
[logr](https://github.com/go-logr/logr) is a logging API for Go. It provides a simple interface for logging under which there is an actual logger that implements the `logr` interface.

[Zap](https://github.com/uber-go/zap) is a log library that implements `logr` interface.

[SDK-generated operators](https://sdk.operatorframework.io/docs/building-operators/golang/references/logging/) use the logr interface to log. Operator SDK by default uses a [zap-based logger](https://pkg.go.dev/sigs.k8s.io/controller-runtime#section-readme) that is ready for production use. The default verbosity is set to `info` level.

_logr_ defines logger's verbosity levels numerically. To write log lines that are more verbose, `logr.Logger` has a [V()](https://pkg.go.dev/github.com/go-logr/logr#hdr-Verbosity) method. The higher the V-level of a log line, the less critical it is considered.
Level `V(0)` is the default, and `logger.V(0).Info()` has the same meaning as `logger.Info()`.

Levels in _logr_ correspond to [custom debug levels](https://pkg.go.dev/go.uber.org/zap/zapcore#Level) in _Zap_. Any given level in logr is represented by its inverse in zap (zapLevel = -1*logrLevel).
Thus, in _Zap_, higher levels are more important.

For example: _logr_ V(2) is equivalent to log level -2 in Zap, while _logr_ V(1) is equivalent to debug level -1 in _Zap_.

**To summarize:**

|Zap logging priority  | Zap enum     | logr                              |
|---------------------:| ------------ | --------------------------------- |
| -1                   | debug        | `.V(1).Info(...)`                 |
|  0                   | info         | `.V(0).Info(...)` or `.Info(...)` |
|  1                   | warn         | N.A.                              |
|  2                   | error        | `.Error(...)`                     |
|  3                   | dpanic       | N.A.                              |
|  4                   | panic        | N.A.                              |
|  5                   | fatal        | N.A.                              |


