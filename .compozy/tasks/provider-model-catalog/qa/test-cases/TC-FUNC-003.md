# TC-FUNC-003: Builtin Source Converts Defaults to Priority-10 Rows

**Priority:** P1
**Type:** Functional
**Module:** `internal/modelcatalog` (`builtin` source)
**Requirement:** TechSpec Source Implementations.
**Status:** Not Run

## Objective

Verify the `builtin` source emits source rows with priority 10, supports offline first-run, and never wins against config or live sources.

## Preconditions

- [ ] Fresh `AGH_HOME` with no overrides for built-in providers.
- [ ] Network disabled (no `models.dev`, no live discovery).

## Test Steps

1. **Boot daemon offline.**
   - Command: `agh daemon start --foreground` with `AGH_DISABLE_OUTBOUND=1` (or stubbed transport).
   - **Expected:** Daemon starts; no errors logged that block startup.
2. **List catalog for a built-in provider (e.g. `codex`).**
   - Command: `agh provider models list codex -o json`.
   - **Expected:** Models present with `sources[0].source_id="builtin"` and `priority=10`; `availability_state="unknown"`.
3. **Add a config curated model that overrides display name.**
   - Update `config.toml` with curated metadata for the same `model_id`.
   - **Expected:** Merged projection shows the config-source `display_name` because priority 120 > 10; builtin row remains addressable as a separate source via `--source builtin`.
4. **Disable the builtin source via internal API.**
   - Programmatically remove builtin source registration in tests.
   - **Expected:** Catalog falls back to remaining sources without panicking; no orphan rows remain in `model_catalog_rows` for the removed source after replace.

## Audit Coverage

- C6 task tree (Task 03).
- SI-13 (partial-source success).

## Pass Criteria

- Builtin rows appear at priority 10.
- Config wins on conflict; builtin survives as second source.
- Removing builtin source does not corrupt rows.

## Failure Criteria

- Builtin priority differs from 10.
- Builtin overrides higher-priority sources.
- Daemon panics or fails to boot offline.
