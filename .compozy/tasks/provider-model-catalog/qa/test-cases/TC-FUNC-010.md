# TC-FUNC-010: ACP `session/set_config_option` Precedence

**Priority:** P0
**Type:** Functional
**Module:** `internal/acp` (Driver.applySessionModel)
**Requirement:** TechSpec ACP Session Config Options, SI-7.
**Status:** Not Run

## Objective

Verify the upgraded SDK driver prefers `session/set_config_option` for model and reasoning effort and only falls back to `session/set_model` when no matching config option exists. Reasoning never sent when no matching control is advertised.

## Preconditions

- [ ] `coder/acp-go-sdk@v0.12.2` upgraded.
- [ ] ACP fake driver fixtures expose `configOptions` for `model` and `reasoning_effort` (and the documented synonyms).

## Test Steps

1. **`session/new` advertises a `model` config option matching the requested model.**
   - **Expected:** Driver issues `session/set_config_option` with `id="model"`, `value=<model>`; never invokes `session/set_model` for this case.
2. **`session/new` advertises a reasoning option.**
   - **Expected:** Driver applies reasoning via `session/set_config_option`; legacy `set_model` not invoked.
3. **`config_option_update` event arrives mid-session.**
   - **Expected:** Driver updates session state; HTTP/UDS session capability surfaces reflect new options on next read.
4. **No matching config option present, but legacy model state advertises the model.**
   - **Expected:** Driver falls back to `session/set_model`; debug log notes fallback reason.
5. **Neither config option nor legacy model state.**
   - **Expected:** Driver does not send any model mutation; reasoning effort is silently skipped (SI-7); session creation succeeds with default state.
6. **Conservative ID matching.**
   - Stub option ID `model_v2` (not in known list).
   - **Expected:** Driver does not assume it is a model option; treats as opaque; falls back as in step 5 if no exact `model` ID.

## Audit Coverage

- C6 task tree (Task 06).
- SI-7.

## Pass Criteria

- Steps 1-3 use `session/set_config_option`.
- Step 4 falls back to `session/set_model`.
- Step 5 sends no mutation.
- Step 6 never invents reasoning levels from `supports_reasoning=true`.

## Failure Criteria

- Driver invokes `session/set_model` when a matching config option exists.
- Reasoning effort fired without an advertised control.
- Unknown option IDs treated as model option.
