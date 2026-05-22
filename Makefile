GO ?= go
DOCKER ?= docker
GO_IMAGE ?= golang:1.26.3
GOVULNCHECK ?= govulncheck
GOVULNCHECK_VERSION ?= latest
GOVULNCHECK_PACKAGE := golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: up down test test-container test-unit test-contract test-integration deps-check docs-comments-check write-inventory-check openapi-check install-govulncheck vulncheck vulncheck-container security security-host security-container

up:
	docker compose up -d

down:
	docker compose down

test: deps-check write-inventory-check openapi-check test-unit test-contract test-integration

test-container:
	$(DOCKER) run --rm \
		-v "$(CURDIR)":/workspace \
		-w /workspace \
		-e GOCACHE=/tmp/go-cache \
		-e GOPATH=/tmp/go \
		$(GO_IMAGE) \
		sh -lc 'export PATH=/usr/local/go/bin:$$PATH; make test GO=go'

test-unit:
	$(GO) test ./... -run 'Test(UUID|WithRetry|Health|Sentinel|States|Checksum)'

test-contract:
	$(GO) test ./... -run 'Test(Readme|AllowedModules|HealthRoutes)'

test-integration:
	$(GO) test ./... -run 'TestWithRetrySerializableRetries'

deps-check:
	$(GO) test ./... -run TestAllowedModules

docs-comments-check:
	node scripts/check_doc_comment_quality.mjs

write-inventory-check:
	node scripts/check_external_write_inventory.mjs

openapi-check:
	node scripts/generate_openapi_spec.mjs --check

install-govulncheck:
	$(GO) install $(GOVULNCHECK_PACKAGE)

vulncheck:
	@if command -v "$(GO)" >/dev/null 2>&1; then \
		if command -v "$(GOVULNCHECK)" >/dev/null 2>&1; then \
			$(GOVULNCHECK) ./...; \
		else \
			$(GO) run $(GOVULNCHECK_PACKAGE) ./...; \
		fi; \
	else \
		$(MAKE) vulncheck-container; \
	fi

vulncheck-container:
	$(DOCKER) run --rm \
		-v "$(CURDIR)":/workspace \
		-w /workspace \
		-e GOCACHE=/tmp/go-cache \
		-e GOPATH=/tmp/go \
		$(GO_IMAGE) \
		sh -lc 'export PATH=/usr/local/go/bin:$$PATH; go run $(GOVULNCHECK_PACKAGE) ./...'

security:
	@if command -v "$(GO)" >/dev/null 2>&1; then \
		if $(MAKE) security-host; then \
			exit 0; \
		fi; \
		echo "Host Go security check failed; retrying in $(GO_IMAGE)."; \
	fi; \
	$(MAKE) security-container

security-host: deps-check vulncheck

security-container:
	$(DOCKER) run --rm \
		-v "$(CURDIR)":/workspace \
		-w /workspace \
		-e GOCACHE=/tmp/go-cache \
		-e GOPATH=/tmp/go \
		$(GO_IMAGE) \
		sh -lc 'export PATH=/usr/local/go/bin:$$PATH; make security-host GO=go'
