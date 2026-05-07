# TC-FUNC-002: Provider Config - Curated Validation Rules

**Priority:** P1
**Type:** Functional
**Module:** `internal/config`
**Requirement:** TechSpec Config Lifecycle.
**Status:** Not Run

## Objective

Verify the nested `[providers.<id>.models]` block enforces the documented validation rules and accepts manual default models.

## Preconditions

- [ ] Fresh `AGH_HOME`.
- [ ] Daemon binary built from current branch.

## Test Steps

1. **Manual default model outside curated list is accepted.**
   - Input: `[providers.codex.models] default = "manual-gpt-9000"` with empty `curated`.
   - **Expected:** Validation succeeds; `agh provider models list codex -o json` includes `manual-gpt-9000` only when sources later report it; manual selection at session creation succeeds.
2. **Duplicate curated `id` is rejected.**
   - Input: two `[[providers.codex.models.curated]]` entries with `id = "gpt-5.4"`.
   - **Expected:** Error references both occurrences.
3. **Blank reasoning effort is rejected.**
   - Input: `reasoning_efforts = ["high", ""]`.
   - **Expected:** Error references the empty entry.
4. **`default_reasoning_effort` must be present in `reasoning_efforts`.**
   - Input: `reasoning_efforts = ["low", "medium"]`, `default_reasoning_effort = "high"`.
   - **Expected:** Error references the curated entry's effort path.
5. **`[model_catalog.sources.models_dev]` defaults populate.**
   - Input: omit the section entirely.
   - **Expected:** `agh config show` resolves `enabled=true`, `endpoint="https://models.dev/api.json"`, `ttl="24h"`, `timeout="10s"`.
6. **`models.discovery.command` and `.endpoint` are mutually exclusive when both set without adapter override.**
   - Input: `[providers.openclaw.models.discovery] command = "x" endpoint = "https://"`.
   - **Expected:** Error states only one of the two is allowed unless the provider adapter documents both.

## Audit Coverage

- C6 task tree (Task 01, Task 03 sources, Task 05 daemon wiring).
- SI-6 (manual model entry remains valid).

## Pass Criteria

- All validation cases produce the documented error or success.
- Defaults appear when omitted.

## Failure Criteria

- Any blank/duplicate/invalid combination is silently accepted.
- Defaults differ from the TechSpec values.
