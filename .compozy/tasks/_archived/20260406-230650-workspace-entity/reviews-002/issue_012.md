---
status: resolved
file: internal/httpapi/handlers_test.go
line: 218
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCM,comment:PRRC_kwDOR5y4QM61T6Hl
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Stop concatenating temp paths into JSON literals.**

These bodies embed filesystem paths directly into JSON strings. That works on POSIX temp paths, but a backslash-containing path makes the request invalid JSON before the handler even runs. Please marshal a request struct/map instead so the tests stay path/OS agnostic.



Also applies to: 347-348, 404-404

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/httpapi/handlers_test.go` around lines 217 - 218, The test
constructs JSON by concatenating temp paths into a string (see the body variable
near performRequest(t, engine, http.MethodPost, "/api/workspaces")), which
breaks on Windows; replace the raw string concat with marshaling a struct/map
using encoding/json (e.g., build a request value with fields root_dir, name,
add_dirs, default_agent and call json.Marshal) before calling performRequest;
apply the same change to the other occurrences flagged (around lines 347-348 and
404) to keep tests OS-agnostic.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The flagged tests interpolate temp paths directly into JSON string literals.
  - That is fragile on platforms where path separators or escaping rules make the literal invalid JSON before the handler sees it.
  - I will marshal request payloads from Go structs/maps instead of concatenating JSON strings.
