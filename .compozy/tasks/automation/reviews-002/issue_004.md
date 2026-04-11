---
status: resolved
file: internal/automation/manager.go
line: 316
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TZaD,comment:PRRC_kwDOR5y4QM623-TL
---

# Issue 004: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap constructor/build failures with operation context.**

These raw returns lose the failing phase once the error bubbles out of `New` or `Start`. Adding context here makes startup failures much easier to diagnose.



<details>
<summary>Suggested fix</summary>

```diff
 dispatcher, err := NewDispatcher(options.sessions, options.store, dispatcherOpts...)
 if err != nil {
-	return nil, err
+	return nil, fmt.Errorf("automation: construct dispatcher: %w", err)
 }
```

```diff
 scheduler, err := NewScheduler(m.dispatcher, schedulerOpts...)
 if err != nil {
-	return nil, nil, err
+	return nil, nil, fmt.Errorf("automation: construct scheduler: %w", err)
 }
 ...
 triggerEngine, err := NewTriggerEngine(m.dispatcher, triggerOpts...)
 if err != nil {
-	return nil, nil, errors.Join(err, m.shutdownRuntimeComponent(ctx, "scheduler", scheduler))
+	return nil, nil, errors.Join(
+		fmt.Errorf("automation: construct trigger engine: %w", err),
+		m.shutdownRuntimeComponent(ctx, "scheduler", scheduler),
+	)
 }
```
</details>

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`."


Also applies to: 1117-1129

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/manager.go` around lines 314 - 316, The NewDispatcher
constructor error is returned raw; wrap it with context before returning so
failures indicate the failing phase. Replace the bare "return nil, err" after
the NewDispatcher(...) call with a wrapped error like fmt.Errorf("create
dispatcher: %w", err). Apply the same pattern to other constructor/startup error
returns in this file (the block around lines 1117-1129) so functions such as
NewDispatcher and the manager New/Start paths return errors wrapped with
descriptive context using fmt.Errorf("%s: %w", ...).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `New()` and `buildRuntimes()` still return some constructor/startup failures without phase context, notably the raw `NewDispatcher()` return and the scheduler/trigger-engine construction path.
  - When those errors bubble out of manager startup, the failing phase is unclear, which makes diagnosis harder than necessary.
  - Fix approach: wrap the relevant constructor/startup failures with explicit operation context and add a focused manager test that exercises a reachable wrapped startup failure.
  - Resolution: wrapped the manager constructor/startup phases with explicit context, added a manager startup regression test for wrapped sync failures, and verified with focused `go test` runs plus `make verify`.
