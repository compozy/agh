MAGE ?= $(shell command -v mage 2>/dev/null)

ifeq ($(strip $(MAGE)),)
MAGE_RUN = go run github.com/magefile/mage@v1.15.0
else
MAGE_RUN = $(MAGE)
endif

.PHONY: deps fmt lint test test-integration codegen codegen-check build verify help

deps:
	@$(MAGE_RUN) deps

fmt:
	@$(MAGE_RUN) fmt

lint:
	@$(MAGE_RUN) lint

test:
	@$(MAGE_RUN) test

test-integration:
	@$(MAGE_RUN) testIntegration

codegen:
	@$(MAGE_RUN) codegen

codegen-check:
	@$(MAGE_RUN) codegenCheck

build:
	@$(MAGE_RUN) build

verify:
	@$(MAGE_RUN) verify

help:
	@$(MAGE_RUN) -l

# Documentation Site
.PHONY: site-dev site-build cli-docs

site-dev:
	@cd packages/site && bun run dev

site-build:
	@cd packages/site && bun run build

cli-docs:
	@go run ./cmd/agh doc --output-dir packages/site/content/runtime/cli-reference

# Web UI
.PHONY: web-dev web-build web-lint web-fmt web-typecheck web-test

web-dev:
	@cd web && bun run dev

web-build:
	@cd web && bun run build

web-lint:
	@cd web && bun run lint

web-fmt:
	@cd web && bun run format

web-typecheck:
	@cd web && bun run typecheck

web-test:
	@cd web && bun run test
