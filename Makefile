# Image URL to use all building/pushing image targets
IMG ?= qalisa/push-github-secrets-operator:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development Setup

.PHONY: setup-dev
setup-dev: install-go install-kubebuilder install-tools ## Install all development dependencies

.PHONY: install-go
install-go: ## Install Go using brew
	brew install go

.PHONY: install-kubebuilder
install-kubebuilder: install-go ## Install Kubebuilder
	curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$$(go env GOOS)/$$(go env GOARCH) && \
	chmod +x kubebuilder && \
	sudo mv kubebuilder /usr/local/bin/

.PHONY: install-tools
install-tools: ## Install required tools using brew
	brew install kind golangci-lint helm

.PHONY: lint
lint: ## Run golangci-lint
	cd src && golangci-lint run

##@ Development

.PHONY: generate-all
generate-all: generate-helpers generate-crds

.PHONY: generate-crds
generate-crds: controller-gen ## Generate CRDs (only run this when API changes)
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./src/..." output:crd:artifacts:config=helm-charts/push-github-secrets-operator/crds

.PHONY: generate-helpers
generate-helpers: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="./src/hack/boilerplate.go.txt" paths="./src/..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	cd src && go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	cd src && go vet ./...

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary.
	cd src && go build -o bin/manager cmd/main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	@if [ ! -f src/.env ]; then \
		echo "Error: src/.env file is required" >&2; \
		exit 1; \
	fi
	@if ! grep -q "GITHUB_APP_ID=" src/.env || \
		! grep -q "GITHUB_INSTALLATION_ID=" src/.env || \
		! grep -q "GITHUB_PRIVATE_KEY_PATH=" src/.env; then \
		echo "Error: .env must contain GITHUB_APP_ID, GITHUB_INSTALLATION_ID, and GITHUB_PRIVATE_KEY_PATH" >&2; \
		exit 1; \
	fi
	source src/.env && cd src && go run ./cmd/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	cd src && docker build -t ${IMG} .

.PHONY: kind-create
kind-create: ## Create kind cluster for local development
	kind create cluster --name operator-dev --config kind-config.yaml || true

.PHONY: kind-delete
kind-delete: ## Delete kind cluster
	kind delete cluster --name operator-dev

.PHONY: docker-load
docker-load: docker-build ## Load docker image into kind cluster.
	kind load docker-image ${IMG} --name operator-dev

.PHONY: docker-push
docker-push: docker-build ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install-crds
install-crds: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f helm-charts/push-github-secrets-operator/crds/

.PHONY: uninstall-crds
uninstall-crds: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete --ignore-not-found=$(ignore-not-found) -f helm-charts/push-github-secrets-operator/crds/

.PHONY: deploy-without-image
deploy-without-image: kind-create generate-all install-crds

.PHONY: deploy
deploy: deploy-without-image docker-load ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	@if [ ! -f src/.env ]; then \
		echo "Error: src/.env file is required" >&2; \
		exit 1; \
	fi
	@if ! grep -q "GITHUB_APP_ID=" src/.env || \
		! grep -q "GITHUB_INSTALLATION_ID=" src/.env || \
		! grep -q "GITHUB_PRIVATE_KEY_PATH=" src/.env; then \
		echo "Error: .env must contain GITHUB_APP_ID, GITHUB_INSTALLATION_ID, and GITHUB_PRIVATE_KEY_PATH" >&2; \
		exit 1; \
	fi
	source src/.env && helm upgrade --install push-github-secrets-operator helm-charts/push-github-secrets-operator \
		--set image.repository=$(shell echo ${IMG} | cut -f1 -d:) \
		--set image.tag=$(shell echo ${IMG} | cut -f2 -d:) \
		--set github.appId="$$GITHUB_APP_ID" \
		--set github.installationId="$$GITHUB_INSTALLATION_ID" \
		--set github.privateKeyPath="$$GITHUB_PRIVATE_KEY_PATH"

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	helm uninstall push-github-secrets-operator

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/src/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.17.2

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: helm-docs
helm-docs: ## Generate Helm chart documentation
	docker run --rm --volume "$(PWD):/helm-docs" -u $(shell id -u) jnorwood/helm-docs:latest

##@ Samples

.PHONY: apply-samples
apply-samples: ## Apply sample CRDs to the cluster
	kubectl apply -f config/samples/
