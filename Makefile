ifndef VERSION
	VERSION = 'v0.0.1'
endif

.PHONY: build
build: build-only

.PHONY: build-only
build-only:
	go mod tidy
	go mod download
	go build -o terraform-provider-contabo_$(VERSION)

.PHONY: doc-preview
doc-preview:
	@echo "Preview your markdown documentation on this page: https://registry.terraform.io/tools/doc-preview"