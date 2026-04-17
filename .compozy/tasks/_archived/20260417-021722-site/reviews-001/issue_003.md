---
status: resolved
file: internal/cli/doc.go
line: 42
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hC_z,comment:PRRC_kwDOR5y4QM64gE4k
---

# Issue 003: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# Check the current implementation of internal/cli/doc.go around lines 22-42
cat -n internal/cli/doc.go | sed -n '15,50p'
```

Repository: compozy/agh

Length of output: 1292

---

🏁 Script executed:

```shell
# Find the docpost package and its Process function
find . -name "*.go" -path "*/docpost/*" | head -20
```

Repository: compozy/agh

Length of output: 127

---

🏁 Script executed:

```shell
# Search for the Process function in docpost
rg -A 5 "func Process" --type go
```

Repository: compozy/agh

Length of output: 503

---

🏁 Script executed:

```shell
# Check the complete doc command implementation to understand context usage
cat -n internal/cli/doc.go
```

Repository: compozy/agh

Length of output: 1661

---

🏁 Script executed:

```shell
# Get the full docpost.Process implementation
cat -n internal/cli/docpost/docpost.go
```

Repository: compozy/agh

Length of output: 16334

---

🏁 Script executed:

```shell
# Check doc.GenMarkdownTree signature from cobra
rg "GenMarkdownTree" --type go -B 2 -A 10
```

Repository: compozy/agh

Length of output: 712

---

🏁 Script executed:

```shell
# Check if there are other context usages in the CLI package for reference
rg "cmd.Context()" --type go
```

Repository: compozy/agh

Length of output: 10232

---



</details>

**Propagate `cmd.Context()` through doc generation pipeline.**

The doc generation path performs filesystem-heavy operations (directory creation, file reads/writes, directory walks) but does not carry context, preventing cancellation and deadline control. Per coding guidelines, context must be the first argument for functions crossing runtime boundaries.

Update `docpost.Process()` to accept context as the first parameter and propagate `cmd.Context()` from the command handler.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/doc.go` around lines 22 - 42, The RunE handler should propagate
cmd.Context() into the doc generation pipeline: change the call site in RunE to
pass cmd.Context() as the first argument to docpost.Process and update the
docpost.Process signature to accept context.Context as its first parameter
(i.e., func Process(ctx context.Context, srcDir, outDir string) error). Also
update any internal calls inside the docpost package that call Process (and any
helper functions used by Process) to accept and forward the context, ensuring
all filesystem-heavy operations (in docpost.Process and its helpers) use the
provided ctx for cancellation and deadline checks.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The CLI entrypoint has a request-scoped context (`cmd.Context()`), but the doc post-processing package drops that context at the package boundary.
  - Root cause: `docpost.Process` and its heavy filesystem helpers do not accept a context, so cancellation and deadlines cannot stop long directory reads/walks between phases.
  - Fix plan: change `docpost.Process` to accept `context.Context`, pass `cmd.Context()` from `newDocCommand()`, thread the context through the read/clean/write phases, and add a canceled-context regression test.
  - Resolution: `docpost.Process` and its heavy helpers now accept `context.Context`, `newDocCommand()` forwards `cmd.Context()`, and `TestProcess_StopsWhenContextCanceled` locks in the cancellation path.
  - Verification: `go test ./internal/cli/...` passed.
