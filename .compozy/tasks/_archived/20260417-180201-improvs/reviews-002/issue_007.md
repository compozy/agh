---
status: resolved
file: internal/tools/tool_test.go
line: 171
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57znpJ,comment:PRRC_kwDOR5y4QM645il_
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify whether ToolSource exposes sentinel/typed errors suitable for errors.Is/errors.As.
rg -nP --type=go -C3 '\btype\s+ToolSource\b|\bfunc\s+\(.*ToolSource.*\)\s+(Validate|MarshalText)\s*\('
rg -nP --type=go -C3 '\bvar\s+Err[A-Za-z0-9_]*\b|errors\.New\(|fmt\.Errorf\(".*%w'
rg -nP --type=go -C3 'marshal tool source|invalid tool source|errors\.Is\(|errors\.As\('
```

Repository: compozy/agh

Length of output: 50369

---

🏁 Script executed:

```shell
# Check lines around 170-171 in the test file for context
sed -n '160,180p' internal/tools/tool_test.go

# Search specifically for error definitions in tool.go
rg -n "^var\s+Err|^type.*Error" internal/tools/tool.go
```

Repository: compozy/agh

Length of output: 756

---

🏁 Script executed:

```shell
# Look at the entire internal/tools/tool.go file to understand error handling approach
wc -l internal/tools/tool.go

# Check the full tool.go file in sections
head -100 internal/tools/tool.go

# Search for all error returns in tool.go
rg -n "error" internal/tools/tool.go | head -20
```

Repository: compozy/agh

Length of output: 3063

---



</details>

**String-based error matching violates project error-handling guidelines.**

`strings.Contains(err.Error(), ...)` is brittle and violates the project's error-matching rule requiring `errors.Is()`/`errors.As()`. However, the current production code does not expose a sentinel or typed error from `Validate()` or `MarshalText()` — both return dynamically constructed errors via `fmt.Errorf()`.

To fix this test properly, first define a sentinel error in `internal/tools/tool.go` (e.g., `var ErrInvalidToolSource = errors.New("tools: invalid tool source")`), then update `Validate()` to wrap that sentinel, and update the test to use `errors.Is()` instead of string matching.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/tools/tool_test.go` around lines 170 - 171, Define a sentinel error
var ErrInvalidToolSource = errors.New("tools: invalid tool source") and update
the Validate() (and any place that constructs the current fmt.Errorf error for
invalid tool source) to wrap that sentinel using fmt.Errorf("%w: ...",
ErrInvalidToolSource) so callers can detect the condition; ensure MarshalText()
propagates or wraps the same sentinel when it fails due to invalid source. Then
update the test (the code calling ToolSource(42).MarshalText()) to use
errors.Is(err, ErrInvalidToolSource) instead of strings.Contains(err.Error(),
"...") to assert the error type.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: invalid `ToolSource` failures are constructed as ad hoc formatted errors, which forces callers and tests to rely on string matching instead of `errors.Is` as required by project error-handling rules.
- Fix plan: introduce a sentinel invalid-source error, wrap it from `Validate` and other invalid-source paths, and update the test to assert the sentinel with `errors.Is`.
- Resolution: `internal/tools/tool.go` now exposes `ErrInvalidToolSource` and wraps it from both `Validate` and `UnmarshalText`, while `TestToolSourceInvalid` now asserts `errors.Is` instead of brittle string matching.
- Verification: `go test ./internal/bundles ./internal/environment/daytona ./internal/extension ./internal/tools` and `make verify` passed on 2026-04-17.
