---
status: resolved
file: internal/acp/process_tree_windows.go
line: 39
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyw5,comment:PRRC_kwDOR5y4QM654NnQ
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**ACP force-exit fallback should not return success by default**

Line 37-Line 39 currently return `nil` without attempting termination. This risks false-positive cleanup in ACP recovery flows on Windows.

Recommend returning a wrapped sentinel error (or implementing best-effort behavior) so callers can distinguish “unsupported/unperformed” from actual success.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/process_tree_windows.go` around lines 35 - 39,
forceManagedProcessGroupExit currently returns nil on Windows which signals
success; change it to return a distinguishable error (or perform a best-effort
termination) so callers can detect unsupported/unperformed behavior. Either (a)
implement a best-effort: check the *exec.Cmd argument, if cmd != nil &&
cmd.Process != nil attempt to terminate/kill the process (respecting the
time.Duration timeout) and return any error from that attempt, or (b) if you
want a simple sentinel, declare a package-level error like var
ErrForceExitUnsupported = errors.New("force-managed-process-group-exit:
unsupported on windows") and return that wrapped with context (fmt.Errorf) from
forceManagedProcessGroupExit; ensure callers can check against
ErrForceExitUnsupported.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Root cause analysis: the current Windows fallback is intentionally a no-op because ACP does not yet have a Windows process-tree implementation for this launcher path. Returning a sentinel error from this helper alone would make normal Windows stop/wait flows report failure even when the direct subprocess exited cleanly, without actually improving child-process cleanup.
- Why not fix here: a correct fix needs a broader Windows tree-management design or updated caller contract that can surface "tree cleanup unsupported" separately from direct-process stop success. That is larger than this scoped batch and cannot be completed safely by changing only this file.
- Resolution: analysis completed; no safe self-contained code change exists for this batch.
