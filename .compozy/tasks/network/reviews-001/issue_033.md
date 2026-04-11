---
status: resolved
file: internal/store/types.go
line: 389
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZv,comment:PRRC_kwDOR5y4QM623eaC
---

# Issue 033: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject or normalize whitespace-padded directions.**

`Validate` trims `Direction` for the enum check, then returns success while leaving the original value untouched. Inputs like `" sent "` will pass validation and can be stored in a form that no longer matches equality-based filters or metrics.


<details>
<summary>Suggested fix</summary>

```diff
 func (e NetworkAuditEntry) Validate() error {
 	if err := requireField(e.SessionID, "network audit session id"); err != nil {
 		return err
 	}
 	if err := requireField(e.Direction, "network audit direction"); err != nil {
 		return err
 	}
-	switch strings.TrimSpace(e.Direction) {
+	direction := strings.TrimSpace(e.Direction)
+	switch direction {
 	case "sent", "received", "rejected":
 	default:
 		return fmt.Errorf("store: network audit direction must be one of %q, %q, %q: %q", "sent", "received", "rejected", e.Direction)
 	}
+	if direction != e.Direction {
+		return fmt.Errorf("store: network audit direction must not contain surrounding whitespace: %q", e.Direction)
+	}
 	if err := requireField(e.Kind, "network audit kind"); err != nil {
 		return err
 	}
 	if err := requireField(e.Space, "network audit space"); err != nil {
 		return err
@@
-	if strings.TrimSpace(e.Direction) == "rejected" && strings.TrimSpace(e.Reason) == "" {
+	if direction == "rejected" && strings.TrimSpace(e.Reason) == "" {
 		return fmt.Errorf("store: network audit reason is required when direction is %q", e.Direction)
 	}
 	return nil
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if err := requireField(e.Direction, "network audit direction"); err != nil {
		return err
	}
	direction := strings.TrimSpace(e.Direction)
	switch direction {
	case "sent", "received", "rejected":
	default:
		return fmt.Errorf("store: network audit direction must be one of %q, %q, %q: %q", "sent", "received", "rejected", e.Direction)
	}
	if direction != e.Direction {
		return fmt.Errorf("store: network audit direction must not contain surrounding whitespace: %q", e.Direction)
	}
	if err := requireField(e.Kind, "network audit kind"); err != nil {
		return err
	}
	if err := requireField(e.Space, "network audit space"); err != nil {
		return err
	}
	if err := requireField(e.PeerFrom, "network audit peer_from"); err != nil {
		return err
	}
	if err := requireField(e.MessageID, "network audit message id"); err != nil {
		return err
	}
	if e.Size < 0 {
		return fmt.Errorf("store: network audit size must be zero or positive: %d", e.Size)
	}
	if direction == "rejected" && strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("store: network audit reason is required when direction is %q", e.Direction)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/types.go` around lines 364 - 389, The validation currently
trims e.Direction only for the switch but leaves the original value unchanged;
update the Validate function to normalize e.Direction (e.g., set e.Direction =
strings.TrimSpace(e.Direction) and optionally strings.ToLower on it) before
performing the enum switch and subsequent checks so stored values are normalized
and downstream equality filters work; also use the normalized value when
checking the "rejected" branch for requiring Reason (e.g., compare the
trimmed/lowercased variable instead of re-trimming e.Direction inline).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `NetworkAuditEntry.Validate` trims `Direction` only for the enum switch, then returns success while leaving the original value untouched. Inputs such as `" sent "` currently validate and can be stored in a form that breaks equality-based filtering. The fix is to reject surrounding whitespace explicitly and to reuse the normalized value for the `"rejected"`-reason check. Because no in-scope store test file exists for this behavior, I expect to use the existing `internal/store/globaldb/global_db_network_audit_test.go` regression suite as a minimal out-of-scope test touch and will keep that change documented and constrained.
  Resolved by rejecting whitespace-padded directions in `internal/store/types.go` and by adding a minimal regression to `internal/store/globaldb/global_db_network_audit_test.go`. Verified with package tests and a clean `make verify`.
