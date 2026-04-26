---
status: resolved
file: internal/cli/spawn.go
line: 160
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsg,comment:PRRC_kwDOR5y4QM67YHC6
---

# Issue 029: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Rename this field or render an actual TTL.**

Line 160 formats `TTLExpiresAt`, so the human output is showing an expiry timestamp under the label `TTL`. That is misleading when someone reads the CLI output.

<details>
<summary>Suggested fix</summary>

```diff
-					{Label: "TTL", Value: stringOrDash(formatTimePtr(record.Lineage.TTLExpiresAt))},
+					{Label: "Expires", Value: stringOrDash(formatTimePtr(record.Lineage.TTLExpiresAt))},
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
				renderHumanSection("Spawn", []keyValue{
					{Label: "Session", Value: stringOrDash(record.Session.ID)},
					{Label: "Agent", Value: stringOrDash(record.Session.AgentName)},
					{Label: "Provider", Value: stringOrDash(record.Session.Provider)},
					{Label: "Workspace", Value: stringOrDash(record.Session.WorkspaceID)},
					{Label: "Parent", Value: stringOrDash(record.Lineage.ParentSessionID)},
					{Label: "Root", Value: stringOrDash(record.Lineage.RootSessionID)},
					{Label: "Depth", Value: fmt.Sprintf("%d", record.Lineage.SpawnDepth)},
					{Label: "Role", Value: stringOrDash(record.Lineage.SpawnRole)},
					{Label: "Expires", Value: stringOrDash(formatTimePtr(record.Lineage.TTLExpiresAt))},
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/spawn.go` around lines 151 - 160, The "TTL" label is misleading
because renderHumanSection is printing record.Lineage.TTLExpiresAt (a timestamp)
via formatTimePtr; update the UI to either rename the label to "TTL Expires" (or
"Expires At") or compute and render the remaining TTL duration instead; locate
the renderHumanSection call in spawn.go and change the key/value pair that
currently uses Label: "TTL" and Value:
stringOrDash(formatTimePtr(record.Lineage.TTLExpiresAt)) to either Label: "TTL
Expires" with the same formatted timestamp or to a computed remaining TTL string
(e.g., subtract time.Now() from record.Lineage.TTLExpiresAt and format) using
formatTimePtr/utility helpers so the label and value accurately reflect expiry
vs remaining time.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The human spawn output labels an expiry timestamp as `TTL`. That reads like a duration but the value is `TTLExpiresAt`.
- Fix: Rename the label to `TTL Expires` while preserving the existing timestamp formatting.
