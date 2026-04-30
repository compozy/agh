---
provider: coderabbit
pr: "85"
round: 2
round_created_at: 2026-04-30T19:49:37.693355Z
status: invalid
file: internal/api/core/tools_test.go
line: 2
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-0IlM,comment:PRRC_kwDOR5y4QM687oqy
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Scope this suite behind an integration build tag.**

This file exercises end-to-end HTTP handler wiring and approval flows; it should be isolated from default unit test runs.

<details>
<summary>Suggested header</summary>

```diff
+//go:build integration
+// +build integration
+
 package core_test
```
</details>

 

As per coding guidelines, "Use integration and E2E build tags (`//go:build integration` or `//go:build e2e`) to separate heavyweight tests from unit tests."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
//go:build integration
// +build integration

package core_test

```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tools_test.go` around lines 1 - 2, This test file is an
end-to-end suite and must be excluded from default unit runs: add a build tag
line above the package declaration (e.g. "//go:build integration") so the file
is compiled only for integration builds; place the tag immediately before
"package core_test" and include the legacy "// +build integration" line if you
support older toolchains, then run `go test` with the integration tag to execute
the suite.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  `internal/api/core/tools_test.go` exercises `BaseHandlers` through an
  in-memory Gin router with stubbed managers and registries. It does not start
  a daemon, UDS socket, HTTP server, subprocess, or external service, and it is
  therefore unit-level handler coverage rather than a heavyweight integration
  suite. Adding `//go:build integration` would remove the core tool handler
  contract coverage from default `make test`, reducing protection for this
  review batch. No code change is required.
