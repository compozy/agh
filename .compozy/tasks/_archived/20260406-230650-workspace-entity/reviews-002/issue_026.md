---
status: resolved
file: internal/udsapi/handlers_test.go
line: 216
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCw,comment:PRRC_kwDOR5y4QM61T6IR
---

# Issue 026: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Stop concatenating temp paths into JSON literals.**

These bodies embed filesystem paths directly into JSON strings. That works on POSIX temp paths, but a backslash-containing path makes the request invalid JSON before the handler even runs. Please marshal a request struct/map instead so the tests stay path/OS agnostic.



Also applies to: 344-345, 401-401

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/udsapi/handlers_test.go` around lines 215 - 216, The test constructs
request bodies by concatenating temp paths into JSON string literals (e.g.,
building body using rootDir and addDir before calling performRequest with
engine, http.MethodPost, "/api/workspaces"), which breaks on Windows
backslashes; instead create a request struct or map with fields like RootDir,
Name, AddDirs, DefaultAgent, populate it with rootDir/addDir, then json.Marshal
that value and pass the resulting bytes to performRequest; apply the same change
to the other test locations that build JSON by string concatenation (the similar
cases in this file).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  These tests build JSON bodies by concatenating temporary filesystem paths into
  string literals. That is brittle for backslashes and quoting and can make the
  request invalid before the handler is exercised. Plan: marshal request structs
  or maps via a shared test helper and reuse it in the affected workspace
  handler tests.
