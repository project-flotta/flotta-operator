# VERSION defines the project version.
# Update this value when you upgrade the version of your project.
VERSION ?= 0.1.0

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
IMAGE_TAG_BASE ?= project-flotta.io/flotta-operator
TEST_IMAGE ?= quay.io/project-flotta/edgedevice:latest
RELEASE_REPO ?= project-flotta/flotta-operator
SKIP_TEST_IMAGE_PULL ?= false

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
HTTP_IMG ?= edge-api:latest

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
# Cluster type - k8s/ocp/all
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
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(Q)$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate-tools:
ifeq (, $(shell which mockery))
	(cd /tmp && go install github.com/vektra/mockery/...@v1.1.2)
endif
ifeq (, $(shell which mockgen))
	(cd /tmp/ && go install github.com/golang/mock/mockgen@v1.6.0)
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
	$(DOCKER) run --rm -v $(PWD):/opt/data/:z docker.io/securego/gosec -exclude-generated /opt/data/...

GO_IMAGE=golang:1.17.8-alpine3.14
GOIMPORTS_IMAGE=golang.org/x/tools/cmd/goimports@latest
FILES_LIST=$(shell ls -d */ | grep -v -E "vendor|tools|test|client|restapi|models|generated")
MODULE_NAME=$(shell head -n 1 go.mod | cut -d '/' -f 3)
imports: ## fix and format go imports
	@# Removes blank lines within import block so that goimports does its magic in a deterministic way
	find $(FILES_LIST) -type f -name "*.go" | xargs -L 1 sed -i '/import (/,/)/{/import (/n;/)/!{/^$$/d}}'
	$(DOCKER) run --rm -v $(CURDIR):$(CURDIR):z -w="$(CURDIR)" $(GO_IMAGE) \
		sh -c 'go install $(GOIMPORTS_IMAGE) && goimports -w -local github.com/project-flotta $(FILES_LIST) && goimports -w -local github.com/project-flotta/$(MODULE_NAME) $(FILES_LIST)'

LINT_IMAGE=golangci/golangci-lint:v1.45.0
lint: ## Check if the go code is properly written, rules are in .golangci.yml 
	$(DOCKER) run --rm -v $(CURDIR):$(CURDIR) -w="$(CURDIR)" $(LINT_IMAGE) sh -c 'golangci-lint run'


ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: ## Run tests.
test: GINKGO_OPTIONS ?= --skip e2e
test: manifests pre-build test-fast

integration-test: ginkgo get-certs
ifeq ($(SKIP_TEST_IMAGE_PULL), false)
	$(DOCKER) pull $(TEST_IMAGE)
endif
	$(GINKGO) -focus=$(FOCUS) run test/e2e

TEST_PACKAGES := ./...
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test-fast: ginkgo
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); $(GINKGO) --cover -output-dir=. -coverprofile=cover.out -v -progress $(GINKGO_OPTIONS) $(TEST_PACKAGES)

test-create-coverage:
	sed -i '/mock_/d' cover.out
	sed -i '/zz_generated/d' cover.out
	go tool cover -func cover.out
	go tool cover --html=cover.out -o coverage.html

test-coverage:
	go tool cover --html=cover.out

vendor:
	go mod tidy -go=1.17
	go mod vendor

get-certs: # Write certificates to /tmp/ folder
	kubectl get secret -n flotta flotta-ca -o go-template='{{ index .data "ca.crt" | base64decode}}' >/tmp/ca.pem
	$(eval REG_SECRET_NAME := $(shell kubectl get secrets -n flotta -l reg-client-ca=true --sort-by=.metadata.creationTimestamp | tail -1 | awk '{print $$1}'))
	kubectl get secret -n flotta $(REG_SECRET_NAME) -o go-template='{{ index .data "client.crt" | base64decode}}' > /tmp/cert.pem
	kubectl get secret -n flotta $(REG_SECRET_NAME) -o go-template='{{ index .data "client.key" | base64decode}}' > /tmp/key.pem

check-certs: # Check cert subject
	openssl x509 -noout -in /tmp/cert.pem --subject

##@ Build
pre-build: ## Generate code, format it and organize imports before executing build
pre-build: generate fmt imports vet

build: pre-build ## Build manager binary.
	go build -mod=vendor -o bin/manager cmd/manager/main.go

fast-build: generate ## Fast build manager binary for local dev.
	go build -mod=vendor -o bin/manager cmd/manager/main.go

run: manifests pre-build ## Run a controller from your host.
	$(Q) kubectl create ns $(FLOTTA_OPERATOR_NAMESPACE) 2> /dev/null || exit 0
	OBC_AUTO_CREATE=false ENABLE_WEBHOOKS=false LOG_LEVEL=debug go run -mod=vendor cmd/manager/main.go

http-api-run: ## Run HTTP API in localhost
	METRICS_ADDR=":8089" go run cmd/httpapi/main.go

docker-build: ## Build docker image with the manager.
	$(DOCKER) build -f build/manager/Dockerfile -t ${IMG} .
	$(DOCKER) build -f build/httpapi/Dockerfile -t ${HTTP_IMG} .


docker-push: ## Push docker image with the manager.
	$(DOCKER) push ${IMG}
	$(DOCKER) push ${HTTP_IMG}

release: ## Release the operator in github releases, tagged by its version.
release: gen-manifests
	gh release create v$(VERSION) --notes "Release v$(VERSION) of Flotta Operator"\
		--repo=$(RELEASE_REPO) --title "Release v$(VERSION)"\
		'$(TMP_ODIR)/ocp-flotta-operator.yaml# Flotta Operator for OCP'\
		'$(TMP_ODIR)/k8s-flotta-operator.yaml# Flotta Operator for kubernetes'
	$(Q)rm -rf $(TMP_ODIR)

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: ## Deploy controller to the K8s cluster specified in ~/.kube/config.
deploy: gen-manifests install-cert-manager
	kubectl apply -f $(TMP_ODIR)/$(TARGET)-flotta-operator.yaml

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
ifeq ($(TARGET), k8s)
	$(KUSTOMIZE) build config/k8s | kubectl delete -f -
else ifeq ($(TARGET), ocp)
	$(KUSTOMIZE) build config/ocp | kubectl delete -f -
endif
	kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.7.1/cert-manager.yaml

$(eval TMP_ODIR := $(shell mktemp -d))
gen-manifests: manifests kustomize ## Generates manifests for deploying the operator into $(TARGET)-flotta-operator.yaml
	$(Q)cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(Q)cd config/edge-api && $(KUSTOMIZE) edit set image edge-api=${HTTP_IMG}
ifneq (,$(filter $(TARGET), k8s all))
	$(Q)$(KUSTOMIZE) build config/k8s > $(TMP_ODIR)/k8s-flotta-operator.yaml
	$(Q)echo -e "\033[92mDeployment file: $(TMP_ODIR)/k8s-flotta-operator.yaml\033[0m"
endif
ifneq (,$(filter $(TARGET), ocp all))
	$(Q)$(KUSTOMIZE) build config/ocp > $(TMP_ODIR)/ocp-flotta-operator.yaml
	$(Q)echo -e "\033[92mDeployment file: $(TMP_ODIR)/ocp-flotta-operator.yaml\033[0m"
endif

	$(Q)cd config/manager && $(KUSTOMIZE) edit set image controller=quay.io/project-flotta/flotta-operator
	$(Q)cd config/edge-api && $(KUSTOMIZE) edit set image edge-api=quay.io/project-flotta/flotta-edge-api

install-router: ## Install openshift router
install-router:
	$(KUBECTL) apply -f https://raw.githubusercontent.com/openshift/router/master/deploy/router_rbac.yaml
	$(KUBECTL) apply -f https://raw.githubusercontent.com/openshift/router/master/deploy/route_crd.yaml
	$(KUBECTL) apply -f https://raw.githubusercontent.com/openshift/router/master/deploy/router.yaml
	$(KUBECTL) wait --for=condition=Ready pods --all -n openshift-ingress --timeout=60s

install-cert-manager: ## Install cert-manager dependency
install-cert-manager:
	$(KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.7.1/cert-manager.yaml
	$(KUBECTL) wait --for=condition=Ready pods --all -n cert-manager --timeout=60s

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@latest)

GINKGO = $(shell pwd)/bin/ginkgo
ginkgo: ## Download ginkgo locally if necessary.
ifeq (, $(shell which ginkgo 2> /dev/null))
	$(call go-install-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.1.3)
endif

CLIENTGEN = $(shell pwd)/bin/client-gen
client-gen: ## Download client-gen locally if necessary.
	$(call go-install-tool,$(CLIENTGEN),k8s.io/code-generator/cmd/client-gen@v0.23.5)

LISTERGEN = $(shell pwd)/bin/lister-gen
lister-gen: ## Download lister-gen locally if necessary.
	$(call go-install-tool,$(LISTERGEN),k8s.io/code-generator/cmd/lister-gen@v0.23.5)

INFORMERGEN = $(shell pwd)/bin/informer-gen
informer-gen: ## Download client-gen locally if necessary.
	$(call go-install-tool,$(INFORMERGEN),k8s.io/code-generator/cmd/informer-gen@v0.23.5)

k8s-client-gen: client-gen lister-gen informer-gen ## Generate typed client for flotta project.
	mkdir gen-tmp
	$(CLIENTGEN) --clientset-name versioned --input-base "" --input github.com/project-flotta/flotta-operator/api//v1alpha1 \
				--output-package github.com/project-flotta/flotta-operator/generated/clientset \
				--output-base gen-tmp --go-header-file hack/boilerplate.go.txt --v 10
	# since the api folder structure has no group folder, the client fails to generate client with group prefix and needs to be renamed
	mv gen-tmp/github.com/project-flotta/flotta-operator/generated/clientset/versioned/typed/v1alpha1/_client.go gen-tmp/github.com/project-flotta/flotta-operator/generated/clientset/versioned/typed/v1alpha1/client.go

	$(LISTERGEN) --input-dirs github.com/project-flotta/flotta-operator/api/v1alpha1 \
				--output-package github.com/project-flotta/flotta-operator/generated/listers \
				--output-base gen-tmp --go-header-file hack/boilerplate.go.txt --v 10

	$(INFORMERGEN) --input-dirs github.com/project-flotta/flotta-operator/api/v1alpha1 \
				--versioned-clientset-package github.com/project-flotta/flotta-operator/generated/clientset/versioned \
				--listers-package github.com/project-flotta/flotta-operator/generated/listers \
				--output-package github.com/project-flotta/flotta-operator/generated/informers \
				--output-base gen-tmp --go-header-file hack/boilerplate.go.txt --v 10
	rm -rf generated
	mv gen-tmp/github.com/project-flotta/flotta-operator/generated ./
	rm -rf gen-tmp

validate-swagger: ## Validate swagger
	$(DOCKER) run --rm -v $(PWD)/.spectral.yaml:/tmp/.spectral.yaml:z -v $(PWD)/swagger.yaml:/tmp/swagger.yaml:z -v $(PWD)/swagger-backend.yaml:/tmp/swagger-backend.yaml:z stoplight/spectral lint --ruleset "/tmp/.spectral.yaml" /tmp/swagger.yaml /tmp/swagger-backend.yaml

generate-agent-install-ostree:
	sed -e "/<CA_PEM>/r /tmp/ca.pem" -e '/<CA_PEM>/d' \
		-e "/<CERT_PEM>/r /tmp/cert.pem" -e '/<CERT_PEM>/d' \
		-e "/<KEY_PEM>/r /tmp/key.pem" -e '/<KEY_PEM>/d' \
		-e "/<CONFIG_TOML>/r hack/config.toml" -e '/<CONFIG_TOML>/d' \
		hack/install-agent-rpm-ostree.sh.template > hack/install-agent-rpm-ostree.sh
	chmod +x hack/install-agent-rpm-ostree.sh

generate-agent-install-dnf:
	sed -e "/<CA_PEM>/r /tmp/ca.pem" -e '/<CA_PEM>/d' \
		-e "/<CERT_PEM>/r /tmp/cert.pem" -e '/<CERT_PEM>/d' \
		-e "/<KEY_PEM>/r /tmp/key.pem" -e '/<KEY_PEM>/d' \
		-e "/<CONFIG_TOML>/r hack/config.toml" -e '/<CONFIG_TOML>/d' \
		hack/install-agent-dnf.sh.template > hack/install-agent-dnf.sh
	chmod +x hack/install-agent-dnf.sh

agent-install-scripts: get-certs generate-agent-install-ostree generate-agent-install-dnf


# go-install-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
