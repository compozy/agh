---
status: resolved
file: magefile.go
line: 542
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575hoz,comment:PRRC_kwDOR5y4QM65B3LW
---

# Issue 003: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Inspect the subprocess helper stack before threading context through it.
sed -n '482,545p' magefile.go
rg -n --type=go '\b(runCommandInDirWithEnv|runRaceEnabledGoCommand)\(' magefile.go magefile_test.go
```

Repository: compozy/agh

Length of output: 2774

---



</details>

**Thread `context.Context` through subprocess helpers and fix error wrapping pattern.**

`runRaceEnabledGoCommand` and `runCommandInDirWithEnv` add runtime-boundary wrappers around `exec.Command` without cancellation paths. Passing a context through the helper stack and switching to `exec.CommandContext` prevents stuck invocations from hanging.

Additionally, the error format `fmt.Errorf("%w %v: %w", errRaceEnabledGoCommand, args, err)` violates the idiomatic error wrapping pattern. Per coding guidelines, use a single wrapped error with context: `fmt.Errorf("context: %w", err)`. Rewrap to `fmt.Errorf("race-enabled go command %v: %w", args, err)` to properly convey args within the message and wrap only the underlying error.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@magefile.go` around lines 537 - 542, The helper runRaceEnabledGoCommand
should accept a context and pass it down to runCommandInDirWithEnv which in turn
must use exec.CommandContext instead of exec.Command so subprocesses are
cancellable; update function signatures (runRaceEnabledGoCommand,
runCommandInDirWithEnv) to include ctx context.Context and thread ctx through
the call chain. Also fix the error wrapping in runRaceEnabledGoCommand to use a
single %w wrap and include the args in the message (e.g.,
fmt.Errorf("race-enabled go command %v: %w", args, err)) so only the underlying
error is wrapped and the args are included as context. Ensure all call sites are
updated to provide a context.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `runCommandInDirWithEnv` currently uses `exec.Command`, so the subprocess helpers do not accept or propagate cancellation at the runtime boundary.
  - `runRaceEnabledGoCommand` also uses a non-idiomatic double-wrap format string (`%w ... %w`) that makes the error chain harder to reason about and is inconsistent with the repo’s error-wrapping rule.
  - Fix approach: thread `context.Context` through `runCommandInDir`, `runCommandInDirWithEnv`, and `runRaceEnabledGoCommand`, switch the helper implementation to `exec.CommandContext`, and rewrap subprocess failures with contextual text plus a single `%w`.
  - Current mage entrypoints do not expose a caller-provided context, so existing call sites will pass `context.Background()` until a higher-level cancellation source exists.
  - Resolved by threading `context.Context` through the mage subprocess helper stack, switching the helper to `exec.CommandContext`, and simplifying the rewrap to a single `%w`.
  - Verified with `go test -tags mage . -run 'TestWithRaceEnabledEnv|TestRunRaceEnabledGoCommand'` and the full `make verify` gate.
