# TC-FUNC-029 — Hook deny / patch behavior

- **Priority:** P1
- **Type:** Functional / hooks
- **Trace:** Task 04, TechSpec Hooks

## Test Steps

1. Pre-call hook returns `decision.deny`.
   - **Expected:** Handle never called; `reason_codes` includes `hook_denied`; payload includes canonical `tool_id`.
2. Pre-call hook patches `request.input` (typed return).
   - **Expected:** Provider receives patched input; non-typed mutations rejected.
3. Pre-call hook attempts to widen authority above ACP ceiling.
   - **Expected:** Ignored; ACP ceiling preserved.
4. Post-call hook redacts result fields.
   - **Expected:** Final result envelope reflects redaction; `redactions` populated.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/hooks -run TestToolHookDenyPatch`
