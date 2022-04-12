# VERSION defines the project version.
# Update this value when you upgrade the version of your project.
VERSION ?= 0.0.1

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
IMAGE_TAG_BASE ?= project-flotta.io/flotta-operator

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
# Cluster type - k8s/ocp/kind
TARGET ?= k8s
# Host name for ingress creation
HOST ?= flotta-operator.srv

# Docker command to use, can be podman
DOCKER ?= docker

# Kubectl command
KUBECTL ?= kubectl

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Default Flotta-operator  namespace
FLOTTA_OPERATOR_NAMESPACE ?= "flotta"

# Set quiet mode by default
Q=@

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(Q)$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate-tools:
ifeq (, $(shell which mockery))
	(cd /tmp && go get github.com/vektra/mockery/.../@v1.1.2)
endif
ifeq (, $(shell which mockgen))
	(cd /tmp/ && go get github.com/golang/mock/mockgen@v1.6.0)
endif
	@exit

generate: generate-tools controller-gen validate-swagger generate-from-swagger ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	go generate ./...

generate-%:
	./hack/generate.sh generate_$(subst -,_,$*)

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

gosec: ## Run gosec locally
	$(DOCKER) run --rm -it -v $(PWD):/opt/data/:z docker.io/securego/gosec -exclude-generated /opt/data/...


ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet test-fast ## Run tests.

integration-test: ginkgo get-certs
	$(DOCKER) pull quay.io/project-flotta/edgedevice
	$(GINKGO) -focus=$(FOCUS) run test/e2e

TEST_PACKAGES := ./...
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test-fast: ginkgo
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); $(GINKGO) --cover -output-dir=. -coverprofile=cover.out -v -progress -skip e2e $(TEST_PACKAGES)

test-create-coverage:
	sed -i '/mock_/d' cover.out
	sed -i '/zz_generated/d' cover.out
	go tool cover -func cover.out
	go tool cover --html=cover.out -o coverage.html

test-coverage:
	go tool cover --html=cover.out

vendor:
	go mod tidy
	go mod vendor

get-certs: # Write certificates to /tmp/ folder
	kubectl get secret -n flotta flotta-ca  -o json | jq '.data."ca.crt"| @base64d' -r >/tmp/ca.pem
	$(eval REG_SECRET_NAME := $(shell kubectl get secrets -n flotta -l reg-client-ca=true --sort-by=.metadata.creationTimestamp | tail -1 | awk '{print $$1}'))
	kubectl -n flotta get secret $(REG_SECRET_NAME) -o json | jq -r '.data."client.crt"| @base64d' > /tmp/cert.pem
	kubectl -n flotta get secret $(REG_SECRET_NAME) -o json | jq -r '.data."client.key"| @base64d' > /tmp/key.pem

check-certs: # Check cert subject
	openssl x509 -noout -in /tmp/cert.pem --subject

##@ Build
build: generate fmt vet ## Build manager binary.
	go build -mod=vendor -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	$(Q) kubectl create ns $(FLOTTA_OPERATOR_NAMESPACE) 2> /dev/null || exit 0
	OBC_AUTO_CREATE=false ENABLE_WEBHOOKS=false LOG_LEVEL=debug go run -mod=vendor ./main.go

docker-build: ## Build docker image with the manager.
	$(DOCKER) build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	$(DOCKER) push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: gen-manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.7.1/cert-manager.yaml
	kubectl wait --for=condition=Ready pods --all -n cert-manager --timeout=60s
	kubectl apply -f $(TMP_ODIR)/$(TARGET)-flotta-operator.yaml
ifeq ($(TARGET), k8s)
	minikube addons enable ingress
endif

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
ifeq ($(TARGET), k8s)
	$(KUSTOMIZE) build config/k8s | kubectl delete -f -
else ifeq ($(TARGET), ocp)
	$(KUSTOMIZE) build config/ocp | kubectl delete -f -
else ifeq ($(TARGET), kind)
	$(KUSTOMIZE) build config/kind | kubectl delete -f -
endif
	kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.7.1/cert-manager.yaml

$(eval TMP_ODIR := $(shell mktemp -d))
gen-manifests: manifests kustomize ## Generates manifests for deploying the operator into $(TARGET)-flotta-operator.yaml
	$(Q)cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
ifeq ($(TARGET), k8s)
	$(Q)sed -i 's/REPLACE_HOSTNAME/$(HOST)/' ./config/k8s/network/ingress.yaml
	$(KUSTOMIZE) build config/k8s > $(TMP_ODIR)/$(TARGET)-flotta-operator.yaml
	$(Q)sed -i 's/$(HOST)/REPLACE_HOSTNAME/' ./config/k8s/network/ingress.yaml
else ifeq ($(TARGET), ocp)
	$(KUSTOMIZE) build config/ocp > $(TMP_ODIR)/$(TARGET)-flotta-operator.yaml
else ifeq ($(TARGET), kind)
	$(KUSTOMIZE) build config/kind > $(TMP_ODIR)/$(TARGET)-flotta-operator.yaml
endif

	$(Q)cd config/manager && $(KUSTOMIZE) edit set image controller=quay.io/flotta-operator/flotta-operator:latest
	$(Q)echo -e "\033[92mDeployment file: $(TMP_ODIR)/$(TARGET)-flotta-operator.yaml\033[0m"

release:
	TARGET=ocp gen-manifests
	TARGET=k8s gen-manifests
	gh release create v$(VERSION) --notes "Release v$(VERSION) of Flotta Operator" --title "Release v$(VERSION)" '$(TMP_ODIR)/ocp-flotta-operator.yaml# Flotta Operator for OCP' '$(TMP_ODIR)/k8s-flotta-operator.yaml# Flotta Operator for kubernetes'
	rm -rf $(TMP_ODIR)

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

GINKGO = $(shell pwd)/bin/ginkgo
ginkgo: ## Download ginkgo locally if necessary.
ifeq (, $(shell which ginkgo 2> /dev/null))
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.1.3)
endif

validate-swagger: ## Validate swagger
	$(DOCKER) run --rm -v $(PWD)/.spectral.yaml:/tmp/.spectral.yaml:z -v $(PWD)/swagger.yaml:/tmp/swagger.yaml:z stoplight/spectral lint --ruleset "/tmp/.spectral.yaml" /tmp/swagger.yaml

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
