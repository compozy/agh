---
status: resolved
file: internal/automation/model/validate.go
line: 355
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564LfU,comment:PRRC_kwDOR5y4QM63o2O8
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate `job.task.network_channel` here as well.**

`JobTaskConfig.Validate()` only checks `owner`, so an invalid `job.task.network_channel` now passes both model and config validation and fails later when the delegated task is materialized. That turns a static misconfiguration into a runtime job failure.

<details>
<summary>Suggested fix</summary>

```diff
+// add github.com/pedronauck/agh/internal/network to imports
 func (c JobTaskConfig) Validate(path string) error {
+	if channel := strings.TrimSpace(c.NetworkChannel); channel != "" {
+		if err := network.ValidateChannel(channel); err != nil {
+			return fmt.Errorf("%s is invalid: %w", nestedPath(path, "network_channel"), err)
+		}
+	}
 	if c.Owner != nil {
 		if err := c.Owner.Validate(nestedPath(path, "owner")); err != nil {
 			return err
 		}
 	}
 	return nil
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/model/validate.go` around lines 349 - 355,
JobTaskConfig.Validate currently only validates c.Owner and ignores
c.NetworkChannel (job.task.network_channel), so invalid network_channel values
slip through; update the JobTaskConfig.Validate(path string) error method to
validate c.NetworkChannel (e.g., call its Validate method or perform the same
checks used elsewhere) using nestedPath(path, "network_channel") and return any
error found; reference the JobTaskConfig type, the Validate method,
c.NetworkChannel (or network_channel field) and nestedPath to locate where to
add the check.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `JobTaskConfig.Validate()` currently validates only `owner` and ignores `network_channel`, so malformed `job.task.network_channel` values survive static validation and fail later during task materialization.
  Root cause: the direct-task config validator never adopted the same network-channel validation used elsewhere in the task/network stack.
  Planned fix: validate the trimmed `network_channel` in `JobTaskConfig.Validate()` and add regression coverage.

## Resolution

- Added `job.task.network_channel` validation in `internal/automation/model/validate.go` so invalid direct-task channels fail during config validation instead of at runtime.
- Kept the validation local to the automation model layer to avoid introducing a package import cycle, and locked it down with a regression case in `internal/automation/validate_test.go`.
