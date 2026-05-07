# TC-INT-006: ACP SDK v0.12.2 - Create / Load / Resume Coverage

**Priority:** P0
**Type:** Integration
**Systems:** `internal/acp` driver + ACP fake fixtures.
**Requirement:** TechSpec ACP Session Config Options, Task 06.
**Status:** Not Run

## Objective

Verify upgrade to `coder/acp-go-sdk@v0.12.2` keeps create/load/resume/mode behavior intact, exposes captured `configOptions`, and propagates `ACPCapsPayload.config_options` / `SessionConfigOptionPayload`.

## Preconditions

- [ ] ACP fake driver with fixtures for `session/new`, `session/load`, `config_option_update`, mode events.

## Test Steps

1. **`session/new` returns `configOptions`.**
   - **Expected:** Driver records options; HTTP/UDS capability payload includes `config_options` with the documented shape.
2. **`session/load` reuses captured options.**
   - **Expected:** No duplicate model mutations.
3. **`session/set_config_option` applied for model and reasoning.**
   - **Expected:** Driver issues the call when matching IDs are advertised; legacy `session/set_model` only used as fallback (TC-FUNC-010 covers exact behavior).
4. **`config_option_update` event mid-session.**
   - **Expected:** Capability payload updated on next read.
5. **Mode/cancellation/error fields renamed in v0.12.2.**
   - **Expected:** Driver compiles and tests prove old behavior intact (existing create/load/resume coverage).
6. **Resume flow.**
   - **Expected:** Resumed session retains `configOptions` from prior load; no new `session/set_*` calls if state matches.

## Audit Coverage

- C5, C6 (Task 06).

## Pass Criteria

- All ACP flows pass on the upgraded SDK.
- `config_options` surface populated.

## Failure Criteria

- ACP driver regresses on create/load/resume.
- `config_options` missing in payload.
