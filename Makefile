GO ?= go

.PHONY: up down test test-unit test-contract test-integration deps-check

up:
	docker compose up -d

down:
	docker compose down

test: deps-check test-unit test-contract test-integration

test-unit:
	$(GO) test ./... -run 'Test(UUID|WithRetry|Health|Sentinel|States|Checksum)'

test-contract:
	$(GO) test ./... -run 'Test(Readme|AllowedModules|HealthRoutes)'

test-integration:
	$(GO) test ./... -run 'TestWithRetrySerializableRetries'

deps-check:
	$(GO) test ./... -run TestAllowedModules
