module github.com/project-flotta/flotta-operator

go 1.16

require (
	github.com/docker/docker v20.10.12+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/errors v0.19.6
	github.com/go-openapi/loads v0.19.5
	github.com/go-openapi/runtime v0.19.20
	github.com/go-openapi/spec v0.19.9
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/swag v0.19.9
	github.com/go-openapi/validate v0.19.10
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kube-object-storage/lib-bucket-provisioner v0.0.0-20210818162813-3eee31c01875
	github.com/onsi/ginkgo/v2 v2.1.3
	github.com/onsi/gomega v1.18.1
	github.com/openshift/api v3.9.0+incompatible
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.15.0
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
)

require (
	github.com/containerd/containerd v1.5.9 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/moby/sys/mount v0.3.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	golang.org/x/sys v0.0.0-20220207234003-57398862261d // indirect
	golang.org/x/tools v0.1.9 // indirect
)
