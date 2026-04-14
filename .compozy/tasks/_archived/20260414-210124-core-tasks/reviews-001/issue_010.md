---
status: resolved
file: internal/api/udsapi/server_test.go
line: 113
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562an2,comment:PRRC_kwDOR5y4QM63mgRG
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**These constructor cases don't isolate the missing dependency.**

The "missing session manager", "missing task service", and "missing observer" branches each omit multiple required options and only assert `err != nil`, so they can pass for the wrong reason if constructor validation order changes. Add the other required deps in each case and assert the specific error you expect.

As per coding guidelines, `**/*_test.go`: MUST have specific error assertions (ErrorContains, ErrorAs).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/server_test.go` around lines 103 - 113, The test cases
calling New(...) in server_test.go currently omit multiple required options so
failures are ambiguous; update each case to include all other required
dependencies except the one under test (use WithHomePaths,
WithSessionManager(stubSessionManager{}), WithTaskService(stubTaskManager{}),
WithObserver(stubObserver{}), WithWorkspaceResolver(stubWorkspaceResolver{}) as
appropriate) so only the intended missing dependency triggers the error, and
replace generic nil-checks with specific assertions (t.ErrorContains or
errors.As) that verify the exact error value or message returned by New for the
missing dependency (reference New, WithSessionManager, WithTaskService,
WithObserver, WithWorkspaceResolver and the stub* types to locate and fix each
case).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the constructor tests omit multiple required dependencies at once and only assert that some error occurred, which makes the failure source ambiguous.
- Fix approach: isolate each missing dependency by supplying the others and assert the exact constructor error string returned by `New`.

## Resolution

- Reworked the constructor coverage into isolated `Should...` cases that provide all unrelated dependencies and assert the specific missing-dependency error message.
- Verified in the final `make verify` run.
