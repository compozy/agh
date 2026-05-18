MAGE ?= $(shell command -v mage 2>/dev/null)

ifeq ($(strip $(MAGE)),)
MAGE_RUN = go run github.com/magefile/mage@v1.15.0
else
MAGE_RUN = $(MAGE)
endif

.PHONY: deps fmt lint test test-integration test-e2e-runtime test-e2e-web test-e2e test-e2e-nightly codegen codegen-check build boundaries verify help bun-lint bun-typecheck bun-test installer-check

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

test-e2e-runtime:
	@$(MAGE_RUN) testE2ERuntime

test-e2e-web:
	@$(MAGE_RUN) testE2EWeb

test-e2e:
	@$(MAGE_RUN) testE2E

test-e2e-nightly:
	@$(MAGE_RUN) testE2ENightly

codegen:
	@$(MAGE_RUN) codegen

codegen-check:
	@$(MAGE_RUN) codegenCheck

build:
	@$(MAGE_RUN) build

boundaries:
	@$(MAGE_RUN) boundaries

verify:
	@$(MAGE_RUN) verify

bun-lint:
	@$(MAGE_RUN) bunLint

bun-typecheck:
	@$(MAGE_RUN) bunTypecheck

bun-test:
	@$(MAGE_RUN) bunTest

installer-check:
	@$(MAGE_RUN) installerCheck

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
	@$(MAGE_RUN) webLint

web-fmt:
	@cd web && bun run format

web-typecheck:
	@bunx turbo run typecheck --filter=./web

web-test:
	@bunx turbo run test --filter=./web
