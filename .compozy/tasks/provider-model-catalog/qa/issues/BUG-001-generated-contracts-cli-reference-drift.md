# BUG-001: Generated Contracts and CLI Reference Drift

**Severity:** Medium
**Priority:** P1
**Type:** Functional
**Status:** Fixed
**Discovered During:** TC-FUNC-015, TC-REG-002
**Reporter:** Codex QA agent
**Created:** 2026-05-07
**Last Updated:** 2026-05-07

## Environment

- **Build:** `2debf0cf` at Task 13 start on branch `fix-migrations`
- **OS:** darwin
- **Browser:** N/A
- **URL / Endpoint:** `make codegen`, `make cli-docs`
- **Bootstrap manifest:** `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/bootstrap-manifest.json`
- **Lab root / runtime home / ports:** lab root `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab`; runtime home `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime`; HTTP port `62444`
- **Live provider/LLM:** not in scope for generated artifact drift

## Summary

Running the generated artifact commands during Task 13 produced tracked diffs in `openapi/agh.json` and 96 generated CLI reference files under `packages/site/content/runtime/cli-reference/`, violating TC-FUNC-015 and TC-REG-002's idempotence expectation.

## Behavioral Impact

- **Operator/User Goal:** Release QA cannot claim generated contracts and generated CLI docs are synchronized without either committing refreshed generated files or proving no drift.
- **Agent Behavior:** Agents relying on committed generated artifacts may inspect stale formatting/output instead of the current generator output.
- **Business Outcome:** Release readiness is degraded because generated artifacts are not reproducible from the committed generator state.
- **Cross-Surface State:** Generated OpenAPI bytes and generated CLI reference docs drift from the repository's current generators.

## Reproduction

```bash
make codegen
make cli-docs
git diff --stat -- openapi/agh.json packages/site/content/runtime/cli-reference
git diff --name-only -- openapi/agh.json packages/site/content/runtime/cli-reference
make codegen-check
```

Observed before fix:

- `openapi/agh.json` changed by formatting-only generator output: `169007` changed lines.
- 96 generated CLI reference files changed, mostly Markdown table alignment and `meta.json` pretty-print formatting.
- `make codegen-check` still exited 0 because OpenAPI check canonicalizes JSON semantically before comparing.
- Semantic comparison of committed vs regenerated OpenAPI with `jq -S` produced no difference.

## Expected

TC-FUNC-015 and TC-REG-002 expect `make codegen` and `make cli-docs` to be idempotent against committed generated artifacts before release QA completion.

## Root Cause

The committed generated artifacts were stale relative to the current generator output. `internal/api/spec.WriteFile` writes canonical OpenAPI with two-space indentation, while the checked-in `openapi/agh.json` was still four-space formatted. `internal/cli/docpost.writeMeta` and Cobra markdown post-processing also generated table/meta formatting that differed from the checked-in CLI reference files.

## Fix

Reran `make codegen`, `make cli-docs`, and the relevant codegen checks until the generated artifacts were idempotent. Final self-review showed no remaining tracked diff under `openapi/agh.json` or `packages/site/content/runtime/cli-reference`, so no generated artifact commit is required from this Task 13 run.

## Verification

- `make codegen-check` after regeneration: passed.
- Semantic OpenAPI comparison via `jq -S`: passed, confirming no contract-shape change.
- `make codegen` idempotence rerun after QA fixes: passed.
- `make cli-docs` idempotence rerun after QA fixes: passed.
- `make codegen-check` after idempotence reruns: passed.
- Final `make verify`: passed via `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/final-make-verify-rerun.log`.
- Final `git diff --stat -- openapi/agh.json packages/site/content/runtime/cli-reference`: no output.

## Impact

- **Users Affected:** Maintainers and agents reading generated contracts/docs from the repository.
- **Frequency:** Always until generated artifacts are refreshed.
- **Workaround:** None. Generated artifacts must be committed in sync with the generators.

## Related

- Test Case: TC-FUNC-015, TC-REG-002
- TechSpec Invariant: Web/Docs Impact
- ADR: ADR-001
- Logs / artifacts:
  - `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-codegen.log`
  - `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-cli-docs.log`
  - `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-codegen-check-after-codegen.log`
  - `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-codegen-idempotence.log`
  - `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-cli-docs-idempotence.log`
  - `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-codegen-check-final-artifacts.log`
