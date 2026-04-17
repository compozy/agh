---
status: resolved
file: internal/testutil/acpmock/driver_binary.go
line: 52
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57ziMu,comment:PRRC_kwDOR5y4QM645avl
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify the unbounded build subprocess and its parallel callers.
rg -n 'exec\.CommandContext\(|DefaultDriverPath\(' --type go internal/testutil/acpmock
rg -n 't\.Parallel\(' --type go internal/testutil/acpmock/driver_test.go internal/testutil/acpmock/fixture_test.go
```

Repository: compozy/agh

Length of output: 3672

---

🏁 Script executed:

```shell
cat -n internal/testutil/acpmock/driver_binary.go
```

Repository: compozy/agh

Length of output: 2597

---



</details>

**Use a timeout context for the subprocess build.**

The `go build` subprocess runs under `context.Background()` with no deadline, so network hangs or stuck module fetches will block the entire parallel test suite indefinitely. Apply a bounded context with timeout (e.g., 30-60 seconds depending on typical build time).

Additionally, this violates the guideline: "Use context.Context as first argument to functions crossing runtime boundaries — avoid context.Background() outside main and focused tests."

<details>
<summary>Code snippet</summary>

```go
cmd := exec.CommandContext(
	context.Background(),
	"go",
	"build",
	"-o",
	outputPath,
	"./internal/testutil/acpmock/cmd/acpmock-driver",
)
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/driver_binary.go` around lines 42 - 52, Replace the
unbounded context.Background() passed to exec.CommandContext with a
timeout-bound context using context.WithTimeout (e.g., 30–60s) so the go build
subprocess cannot hang indefinitely; create ctx, cancel :=
context.WithTimeout(context.Background(), 30*time.Second) and defer cancel(),
then call exec.CommandContext(ctx, "go", "build", "-o", outputPath,
"./internal/testutil/acpmock/cmd/acpmock-driver"); update imports if necessary
to include time and context and keep using the existing cmd variable and
repoRoot handling unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `DefaultDriverPath` currently builds the helper binary with `exec.CommandContext(context.Background(), ...)`, so a stuck `go build` can hang the entire test process indefinitely.
  - This crosses a subprocess/runtime boundary and should use a bounded context according to the repo guidance.
  - Implemented: extracted the build step into `buildDriverBinary(...)` and wrapped the real `go build` call in a timeout-bound context from `DefaultDriverPath`.
  - Regression coverage: added `TestBuildDriverBinaryHonorsContextCancellation`.
  - Verification: `go test ./internal/testutil/acpmock -count=1`; `make verify`.
