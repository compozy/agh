---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/situation/service_test.go
line: 480
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-b,comment:PRRC_kwDOR5y4QM6-VcC6
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Pin the failing reason, not just the sentinel.**

These negative-path tests only use `errors.Is(...)`, so they still pass if the method returns the same sentinel from an earlier, unrelated validation branch. Keep the sentinel check, but also assert a stable substring that identifies the mismatched run/task path and the oversized-bundle path.

 
As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)."


Also applies to: 518-520

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/situation/service_test.go` around lines 477 - 480, The test for
TaskRunPromptOverlayByID only asserts the sentinel error (taskpkg.ErrValidation)
and can mask which validation branch failed; update the assertions to pin the
failing reason by keeping the errors.Is check and additionally assert the error
message contains a stable identifying substring (e.g., "mismatched run" for the
mismatched run path and "oversized bundle" for the oversized-bundle path) so the
test verifies both the sentinel and the specific failure; apply this to the
TaskRunPromptOverlayByID case using mismatchedRun and taskRecord.ID and
similarly update the other failing assertion block at the referenced lines to
use ErrorContains (or errors.As plus message substring) to ensure the exact
validation branch is tested.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the negative-path tests only check the sentinel error, so they can pass even if the wrong validation branch produces the same top-level error class.
- Fix approach: Keep the sentinel checks and add stable message assertions for the mismatched-run and oversized-bundle branches.
