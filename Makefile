ifndef VERSION
	VERSION = 'v0.1.2'
endif

.PHONY: build
build: build-only

.PHONY: build-only
build-only:
	go mod tidy
	go mod download
	go build -o terraform-provider-coderforge_$(VERSION)

.PHONY: build-dev
build-dev:
	go build -o registry.terraform.io/coderforge/coderforge

.PHONY: validate-examples
validate-examples: build-dev
	cd examples/resources/function && TF_CLI_CONFIG_FILE=../../..//.terraformrc terraform init -input=false -upgrade && TF_CLI_CONFIG_FILE=../../..//.terraformrc terraform validate
	cd examples/resources/container_registry && TF_CLI_CONFIG_FILE=../../..//.terraformrc terraform init -input=false -upgrade && TF_CLI_CONFIG_FILE=../../..//.terraformrc terraform validate

.PHONY: hooks-install
hooks-install:
	git config core.hooksPath .githooks

.PHONY: doc-preview
doc-preview:
	@echo "Preview your markdown documentation on this page: https://registry.terraform.io/tools/doc-preview"