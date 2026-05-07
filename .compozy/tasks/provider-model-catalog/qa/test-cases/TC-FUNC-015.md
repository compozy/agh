# TC-FUNC-015: Generated Contracts and Docs Drift Gate

**Priority:** P1
**Type:** Functional
**Module:** Codegen + Docs
**Requirement:** TechSpec Web/Docs Impact, Task 10.
**Status:** Not Run

## Objective

Verify `make codegen` regenerates `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and CLI references; the docs vitest enforces hard-cut copy in `packages/site`.

## Preconditions

- [ ] Working tree clean except QA artifacts.
- [ ] `make` toolchain available.

## Test Steps

1. **Run codegen.**
   - Command: `make codegen`.
   - **Expected:** `git status` shows no diff (committed state already matches generated).
2. **Run codegen-check.**
   - Command: `make codegen-check`.
   - **Expected:** Exit 0.
3. **Run docs vitest.**
   - Command: `cd packages/site && bun run test -- provider-model-catalog-docs`.
   - **Expected:** Suite passes; no flat-field claims (`default_model`, `supported_models`, `supports_reasoning_effort`) in narrative copy outside the hard-cut warning.
4. **CLI docs regenerated.**
   - Command: `make cli-docs`.
   - **Expected:** `packages/site/content/runtime/cli/provider/models/{list,refresh,status}.mdx` reflects current cobra exports.
5. **Inspect MDX sources.**
   - Command: `grep -R "default_model\|supported_models\|supports_reasoning_effort" packages/site/content/runtime`.
   - **Expected:** No matches outside hard-cut warning copy.

## Audit Coverage

- C6 task tree (Task 10).

## Pass Criteria

- Codegen idempotent; docs vitest green.

## Failure Criteria

- Codegen produces diff.
- Docs vitest fails.
- Hard-cut residue in narrative copy.
