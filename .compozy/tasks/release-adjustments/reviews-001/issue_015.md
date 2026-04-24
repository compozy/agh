---
status: resolved
file: internal/store/globaldb/global_db_session.go
line: 642
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1r,comment:PRRC_kwDOR5y4QM67HMW6
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don’t silently clear `activity_json` on serialization failure.**

Returning `""` here makes the write path look successful while dropping the activity payload. Bubble this up as an error from `RegisterSession` / `UpdateSessionState` instead of coercing it to an empty column.


As per coding guidelines, `**/*.go`: Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_session.go` around lines 634 - 642, The
helper sessionLivenessActivityJSON must stop swallowing JSON marshal errors;
change its signature to return (string, error), have it return wrapped errors
using fmt.Errorf("session liveness activity marshal: %w", err) when json.Marshal
fails (keep the nil checks and CloneSessionActivityMeta), and update all callers
such as RegisterSession and UpdateSessionState to propagate and handle the error
(returning it up the write path so the caller can fail instead of writing an
empty activity_json). Ensure all propagated errors are wrapped with contextual
messages per guidelines.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `sessionLivenessActivityJSON` currently returns an empty string on JSON marshal failure, making the write path look successful while dropping runtime activity.
  - The fix is to return `(string, error)`, propagate wrapped errors through `RegisterSession` and `UpdateSessionState`, and add failing-path tests using an unmarshalable `time.Time`.
