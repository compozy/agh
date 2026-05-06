# BUG-004: ACP Mock Diagnostics Dropped Prompt Augmentation

**Severity:** Medium  
**Priority:** P1  
**Type:** Data  
**Status:** Fixed

## Environment

- **Build:** local dev build from task_32 QA execution
- **OS:** macOS, isolated AGH lab from `qa/bootstrap-manifest.json`
- **Browser:** not applicable
- **URL:** daemon memory recall E2E diagnostics
- **Live provider/LLM:** acpmock-backed daemon E2E

## Summary

The memory recall E2E lost the durable-memory preamble in acpmock prompt diagnostics. Matching still worked, but diagnostics recorded only the user text, hiding the actual runtime prompt.

## Behavioral Impact

- **Operator/User Goal:** QA cannot prove memory recall was injected into the model-facing prompt.
- **Agent Behavior:** Diagnostics misrepresent the prompt sent to the agent.
- **Business Outcome:** Memory and context QA can produce false negatives.
- **Cross-Surface State:** Runtime prompt assembly and acpmock diagnostic artifacts diverged.

## Reproduction

```bash
go test -race -parallel=4 -count=1 -tags integration \
  -run '^TestDaemonE2EMemoryRecallUsesCatalogSynthesisWithoutMutatingStoredUserMessage$' ./internal/daemon
```

Observed before the fix:

- Prompt diagnostics contained `remember me` instead of the expected `Relevant durable memory for this turn:` preamble.

## Expected

Matcher normalization may strip augmentation for fixture matching, but diagnostic output must preserve the raw prompt sent by the runtime.

## Root Cause

`internal/testutil/acpmock/cmd/acpmock-driver/main.go` used the same prompt extraction behavior for matching and diagnostics, trimming to the text after `User message:`.

## Fix

The driver now preserves the trimmed last text block for diagnostics, while fixture matching keeps canonicalization in `internal/testutil/acpmock/fixture.go`.

## Verification

- `go test ./internal/testutil/acpmock ./internal/testutil/acpmock/cmd/acpmock-driver -count=1`
- `go test -race -parallel=4 -count=1 -tags integration -run '^TestDaemonE2EMemoryRecallUsesCatalogSynthesisWithoutMutatingStoredUserMessage$' ./internal/daemon`
- Final `make verify`

## Impact

- **Users Affected:** QA engineers and developers reading acpmock diagnostics.
- **Frequency:** Always for augmented prompts.
- **Workaround:** None.

## Related

- Test Case: TC-SEC-001

