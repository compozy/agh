---
status: resolved
file: internal/cli/doc_test.go
line: 27
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hC_Q,comment:PRRC_kwDOR5y4QM64gE32
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Help-output assertion is too broad and can false-positive.**

Checking `strings.Contains(help, "doc")` can match unrelated words. Assert the command token more precisely (e.g., line/column command entry pattern).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/doc_test.go` around lines 25 - 27, The test currently uses
strings.Contains(help, "doc") which can match substrings and produce false
positives; update the assertion to check for the doc command token precisely by
examining the help string lines or using a regex that matches a command entry
(e.g., look for a line that starts with optional whitespace then the literal
"doc" followed by whitespace or a column/description separator) instead of a
simple substring check — locate the test where the variable help is asserted and
replace the strings.Contains check with a line-scan or regex match that ensures
"doc" appears as a standalone command token.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `strings.Contains(help, "doc")` can match unrelated substrings in the help text and can therefore fail for reasons unrelated to the hidden command itself.
  - Root cause: the test uses a broad substring check instead of asserting against the actual command-token shape in the help output.
  - Fix plan: replace the substring assertion with a precise help-line match and keep the test focused on whether `doc` appears as a standalone command entry.
  - Resolution: replaced the broad substring check with a command-entry regex in `TestNewDocCommand_NotInHelp`.
  - Verification: `go test ./internal/cli/...` passed.
