---
status: resolved
file: internal/cli/task_test.go
line: 604
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564Lff,comment:PRRC_kwDOR5y4QM63o2PL
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Fix these bundle assertions to match the current renderers.**

The expectations here drift from the current helper implementations: `taskDetailBundle(...).toon()` only combines task/runs/dependencies, `taskRunBundle(...).human()` does not render `Idempotency Key` or `Result`, and `taskRunListBundle(...).toon()` emits a much smaller schema. As written, this block will fail against `internal/cli/task.go:1013-1043`, `internal/cli/task.go:1045-1089`, and `internal/cli/task.go:1091-1123`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/task_test.go` around lines 580 - 604, The assertions are out of
sync with the current renderers: update the tests to match what
taskDetailBundle(...).toon() actually emits (check for task, runs and
dependencies sections rather than child/event sections), remove expectations for
"Idempotency Key" and "Result" from taskRunBundle(...).human() (keep a generic
check like "Task Run" and at least one run field such as "Status" or "Started
At"), and tighten the taskRunListBundle(...).toon() assertion to the smaller
schema it emits (e.g., assert for a compact task_runs array like
"task_runs[1]{id,status,attempt}" instead of the long field list). Ensure you
modify the three assertions that reference taskDetailBundle().toon(),
taskRunBundle().human(), and taskRunListBundle().toon() accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  The current renderers still match the assertions in this test block. `taskDetailBundle(...).toon()` includes child, dependency, run, and event sections; `taskRunBundle(...).human()` still renders `Idempotency Key` and `Result`; and `taskRunListBundle(...).toon()` still emits the full task-run schema asserted here.
  The review comment appears to reference an older or different renderer shape, not the current implementation under test.

## Resolution

- No code change was made. The current CLI renderers and assertions are already aligned, so this review comment does not reflect the present implementation.
