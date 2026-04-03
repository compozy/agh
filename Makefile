MAGE ?= $(shell command -v mage 2>/dev/null)

ifeq ($(strip $(MAGE)),)
MAGE_RUN = go run github.com/magefile/mage@v1.15.0
else
MAGE_RUN = $(MAGE)
endif

.PHONY: deps fmt lint test test-integration build verify help

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

build:
	@$(MAGE_RUN) build

verify:
	@$(MAGE_RUN) verify

help:
	@$(MAGE_RUN) -l

# Web UI
.PHONY: web-dev web-build web-lint web-fmt web-typecheck web-test

web-dev:
	@bun run dev

web-build:
	@bun run build

web-lint:
	@bun run lint

web-fmt:
	@bun run format

web-typecheck:
	@bun run typecheck

web-test:
	@bun run test
