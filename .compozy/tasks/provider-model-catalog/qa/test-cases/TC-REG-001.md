# TC-REG-001: Hard-Cut Residue Repository Scan

**Priority:** P1
**Type:** Regression
**Surface:** Repository.
**Requirement:** ADR-002, Task 11.1.
**Status:** Not Run

## Objective

Verify no production code or generated artifact references `default_model`, `supported_models`, or `supports_reasoning_effort` outside the documented hard-cut warning copy and historical migration text.

## Preconditions

- [ ] Working tree clean except QA artifacts.

## Test Steps

1. **Repository grep.**
   - Command: `grep -nE "default_model|supported_models|supports_reasoning_effort" -r --include="*.go" --include="*.ts" --include="*.tsx" --include="*.json" --include="*.toml" .`
   - **Expected:** Only known allowlisted matches appear:
     - `internal/modelcatalog/hardcut_residue_test.go` and related tests asserting the residue scan.
     - `packages/site` warning copy (`provider-model-catalog-docs.test.ts`).
     - QA artifacts under `.compozy/tasks/provider-model-catalog/qa/`.
   - No production source under `internal/`, `web/src/`, `cmd/`, `openapi/`, or generated TS/openapi files contain the literal strings.
2. **Generated contracts.**
   - Inspect `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`.
   - **Expected:** No occurrences of the deleted fields.
3. **Web E2E fixtures.**
   - Inspect `web/e2e/fixtures/`.
   - **Expected:** No references to deleted keys.
4. **Site narrative copy.**
   - **Expected:** Only hard-cut warning copy mentions the deleted keys; the docs vitest enforces this.

## Audit Coverage

- C6 (Task 11), C8.

## Pass Criteria

- Grep produces only allowlisted matches.

## Failure Criteria

- Any unexpected reference.
