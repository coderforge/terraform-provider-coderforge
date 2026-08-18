# Copyright (c) 2026 CoderForge.org Ltd.
# Licensed under the MIT License.

VERSION      ?= 2.0.0
BINARY       := terraform-provider-coderforge
HOSTNAME     := registry.terraform.io
NAMESPACE    := coderforge
NAME         := coderforge
OS_ARCH      ?= $(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_DIR  := $(HOME)/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)

.PHONY: help
help:
	@echo "build     Build $(BINARY)_v$(VERSION)"
	@echo "install   Build and install into the local plugin mirror"
	@echo "test      Unit tests; no API required"
	@echo "testacc   Acceptance tests; needs TF_ACC, CODERFORGE_ENDPOINT and CODERFORGE_TOKEN"
	@echo "docs      Regenerate docs/ from the schemas"
	@echo "fmt       gofmt the Go sources and terraform fmt the examples"
	@echo "lint      go vet"
	@echo "check     fmt, lint and test"

.PHONY: build
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)_v$(VERSION)

.PHONY: install
install:
	mkdir -p $(INSTALL_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(INSTALL_DIR)/$(BINARY)_v$(VERSION)
	@echo "Installed into $(INSTALL_DIR)"

.PHONY: test
test:
	go test ./... -timeout 120s

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v -timeout 120m

.PHONY: docs
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate \
		--provider-dir . -provider-name coderforge --rendered-provider-name CoderForge

.PHONY: fmt
fmt:
	gofmt -w internal/ main.go
	terraform fmt -recursive ./examples/

.PHONY: lint
lint:
	go vet ./...

.PHONY: check
check: fmt lint test

.PHONY: clean
clean:
	rm -f $(BINARY)_v* $(BINARY).exe

.PHONY: hooks-install
hooks-install:
	git config core.hooksPath .githooks
