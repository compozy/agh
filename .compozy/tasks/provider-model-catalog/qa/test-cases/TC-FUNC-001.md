# TC-FUNC-001: Provider Config Hard-Cut - Old Keys Rejected

**Priority:** P0
**Type:** Functional
**Module:** `internal/config`
**Requirement:** ADR-002, TechSpec Delete Targets, Task 01.
**Status:** Not Run
**Created:** 2026-05-07
**Last Updated:** 2026-05-07

## Objective

Verify that any `config.toml` containing the deleted flat provider model fields fails validation with deterministic, path-scoped errors and that no compatibility fallback rehydrates the values.

## Preconditions

- [ ] Fresh isolated `AGH_HOME` (no prior config cache).
- [ ] Daemon binary built from current branch.

## Test Steps

1. **Write `config.toml` with the deleted `default_model` key.**
   - Input:
     ```toml
     [providers.codex]
     command = "/bin/true"
     default_model = "gpt-5.4"
     ```
   - **Expected:** `agh config validate` (and daemon boot) returns an error referencing path `providers.codex.default_model` and explicitly stating the key is removed.
2. **Replace with deleted `supported_models` key.**
   - Input:
     ```toml
     [providers.codex]
     command = "/bin/true"
     supported_models = ["gpt-5.4"]
     ```
   - **Expected:** Error references `providers.codex.supported_models`.
3. **Replace with deleted `supports_reasoning_effort` key.**
   - Input:
     ```toml
     [providers.codex]
     command = "/bin/true"
     supports_reasoning_effort = true
     ```
   - **Expected:** Error references `providers.codex.supports_reasoning_effort`.
4. **Confirm new nested shape parses cleanly.**
   - Input:
     ```toml
     [providers.codex]
     command = "/bin/true"
     [providers.codex.models]
     default = "gpt-5.4"
     [[providers.codex.models.curated]]
     id = "gpt-5.4"
     supports_reasoning = true
     reasoning_efforts = ["minimal", "low", "medium", "high", "xhigh"]
     default_reasoning_effort = "medium"
     ```
   - **Expected:** Validation succeeds; daemon starts; `agh provider models list codex -o json` returns rows tagged with `source_id="config"` and priority `120`.

## Negative / Boundary Tests

- Empty curated array with valid `default` → must succeed (manual default model is valid, SI-6).
- `default = ""` → must fail with explicit path `providers.codex.models.default`.
- Curated model `id` blank → must fail.
- `default_reasoning_effort = "extreme"` not in `reasoning_efforts` → must fail.

## Audit Coverage

- C6 task tree (Task 01).
- C8 cross-surface truth: rendered `agh config show` and persisted SQLite catalog row both reflect new shape.
- TechSpec Safety Invariants: SI-6 (manual entry valid), SI-8 (only `internal/modelcatalog.Store` writes catalog rows).

## Pass Criteria

- Steps 1-3 fail with the documented error path; no silent hydrate of legacy fields.
- Step 4 produces catalog rows attributed to the `config` source.
- `agh config show` does not emit any of the deleted keys.

## Failure Criteria

- Any deleted key parses without error.
- Error path lacks the offending key name.
- Catalog row attributes the data to a source other than `config` (priority 120).
