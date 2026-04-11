---
status: resolved
file: internal/api/core/handlers.go
line: 645
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Q88-,comment:PRRC_kwDOR5y4QM6200jQ
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't log the raw home path on resolution failures.**

Line 638 bakes `userHomeDir` into the returned error, and Line 622 logs that error verbatim. On real systems that path commonly embeds the username, so a failing lookup now leaks PII into operational logs. Return a redacted error here and keep the path out of the log payload.

<details>
<summary>🔒 Proposed fix</summary>

```diff
-			if resolveErr != nil {
-				return "", fmt.Errorf("resolve user home directory %q: %w", userHomeDir, resolveErr)
-			}
+			if resolveErr != nil {
+				return "", fmt.Errorf("resolve user home directory: %w", resolveErr)
+			}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/handlers.go` around lines 612 - 645, The returned error
from resolveUserHomeDir must not include the raw userHomeDir string; update
resolveUserHomeDir so that on ResolvePath failure it returns a sanitized error
that omits the original path (e.g., use a generic message like "resolve user
home directory: <redacted>" or wrap resolveErr without embedding userHomeDir)
and keep fallbackUserHomeDir logic unchanged; ensure daemonUserHomeDir and its
logger continue to log the returned error but will no longer receive the raw
path. Reference resolveUserHomeDir, daemonUserHomeDir, fallbackUserHomeDir, and
aghconfig.ResolvePath when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `resolveUserHomeDir` currently formats the `ResolvePath` failure with the raw `userHomeDir` string, and `daemonUserHomeDir` logs that wrapped error verbatim.
- When the lookup value contains a user-specific absolute path, the warning log leaks unnecessary local-path PII even though the path itself is not needed for diagnosis.
- Fix approach: sanitize the returned error by omitting the raw path and strengthen the internal test to assert the failure message stays redacted.
