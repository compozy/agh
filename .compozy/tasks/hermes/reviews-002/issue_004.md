---
status: resolved
file: internal/cli/lifecycle.go
line: 291
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLiV,comment:PRRC_kwDOR5y4QM67SmDZ
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**TOON output drops `daemon_stopped` and `removed`.**

Line 280’s field list omits the uninstall-specific fields that are already present in `lifecycleRecord` and human/JSON output. TOON consumers cannot tell whether the daemon was actually stopped or which artifacts were removed.



<details>
<summary>Suggested fix</summary>

```diff
		toon: func() (string, error) {
			return renderToonObject(
				strings.ToLower(title),
-				[]string{"command", "status", "managed", "manager", "home_dir", "message", "recommendation", "purged"},
+				[]string{"command", "status", "managed", "manager", "home_dir", "message", "recommendation", "daemon_stopped", "removed", "purged"},
				[]string{
					record.Command,
					record.Status,
					fmt.Sprintf("%t", record.Managed),
					record.Manager,
					record.HomeDir,
					record.Message,
					record.Recommendation,
+					fmt.Sprintf("%t", record.DaemonStopped),
+					strings.Join(record.Removed, ", "),
					fmt.Sprintf("%t", record.Purged),
				},
			), nil
		},
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		toon: func() (string, error) {
			return renderToonObject(
				strings.ToLower(title),
				[]string{"command", "status", "managed", "manager", "home_dir", "message", "recommendation", "daemon_stopped", "removed", "purged"},
				[]string{
					record.Command,
					record.Status,
					fmt.Sprintf("%t", record.Managed),
					record.Manager,
					record.HomeDir,
					record.Message,
					record.Recommendation,
					fmt.Sprintf("%t", record.DaemonStopped),
					strings.Join(record.Removed, ", "),
					fmt.Sprintf("%t", record.Purged),
				},
			), nil
		},
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/lifecycle.go` around lines 277 - 291, The TOON renderer inside
the anonymous toon func for renderToonObject is missing the uninstall-specific
fields daemon_stopped and removed; update the field name slice (currently
including "command","status",...,"purged") to also include "daemon_stopped" and
"removed", and add the corresponding values from the lifecycleRecord into the
values slice (e.g., convert record.DaemonStopped to a string via
fmt.Sprintf("%t") and stringify record.Removed appropriately) so
renderToonObject receives the same uninstall fields that human/JSON outputs
expose.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `lifecycleBundle` renders `daemon_stopped` and `removed` in human/JSON paths but the TOON field/value lists only include `purged`, so TOON consumers lose uninstall result details.
- Fix approach: include `daemon_stopped` and `removed` in the TOON schema and values, using the existing `lifecycleRecord` fields.
