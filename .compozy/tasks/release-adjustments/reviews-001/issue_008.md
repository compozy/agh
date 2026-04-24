---
status: resolved
file: internal/config/config.go
line: 809
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1C,comment:PRRC_kwDOR5y4QM67HMWG
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject warning thresholds that exceed the timeout.**

When both `InactivityWarningAfter` and `InactivityTimeout` are set, `warning_after > timeout` is unreachable configuration. The validator currently accepts it, so operators can configure “timeout with no possible prior warning” without feedback.


<details>
<summary>Suggested validation</summary>

```diff
 func (c SessionSupervisionConfig) Validate() error {
     switch {
     case c.ActivityHeartbeatInterval <= 0:
         return fmt.Errorf(
             "session.supervision.activity_heartbeat_interval must be positive: %s",
             c.ActivityHeartbeatInterval,
         )
     case c.ProgressNotifyInterval < 0:
         return fmt.Errorf(
             "session.supervision.progress_notify_interval "+
                 "must be zero or positive: %s",
             c.ProgressNotifyInterval,
         )
     case c.InactivityWarningAfter < 0:
         return fmt.Errorf(
             "session.supervision.inactivity_warning_after "+
                 "must be zero or positive: %s",
             c.InactivityWarningAfter,
         )
     case c.InactivityTimeout < 0:
         return fmt.Errorf("session.supervision.inactivity_timeout must be zero or positive: %s", c.InactivityTimeout)
+    case c.InactivityTimeout > 0 &&
+        c.InactivityWarningAfter > 0 &&
+        c.InactivityWarningAfter > c.InactivityTimeout:
+        return fmt.Errorf(
+            "session.supervision.inactivity_warning_after must be <= session.supervision.inactivity_timeout: %s > %s",
+            c.InactivityWarningAfter,
+            c.InactivityTimeout,
+        )
     case c.TimeoutCancelGrace <= 0:
         return fmt.Errorf("session.supervision.timeout_cancel_grace must be positive: %s", c.TimeoutCancelGrace)
     default:
         return nil
     }
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (c SessionSupervisionConfig) Validate() error {
    switch {
    case c.ActivityHeartbeatInterval <= 0:
        return fmt.Errorf(
            "session.supervision.activity_heartbeat_interval must be positive: %s",
            c.ActivityHeartbeatInterval,
        )
    case c.ProgressNotifyInterval < 0:
        return fmt.Errorf(
            "session.supervision.progress_notify_interval "+
                "must be zero or positive: %s",
            c.ProgressNotifyInterval,
        )
    case c.InactivityWarningAfter < 0:
        return fmt.Errorf(
            "session.supervision.inactivity_warning_after "+
                "must be zero or positive: %s",
            c.InactivityWarningAfter,
        )
    case c.InactivityTimeout < 0:
        return fmt.Errorf("session.supervision.inactivity_timeout must be zero or positive: %s", c.InactivityTimeout)
    case c.InactivityTimeout > 0 &&
        c.InactivityWarningAfter > 0 &&
        c.InactivityWarningAfter > c.InactivityTimeout:
        return fmt.Errorf(
            "session.supervision.inactivity_warning_after must be <= session.supervision.inactivity_timeout: %s > %s",
            c.InactivityWarningAfter,
            c.InactivityTimeout,
        )
    case c.TimeoutCancelGrace <= 0:
        return fmt.Errorf("session.supervision.timeout_cancel_grace must be positive: %s", c.TimeoutCancelGrace)
    default:
        return nil
    }
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/config.go` around lines 783 - 809, The validator in
SessionSupervisionConfig.Validate currently allows InactivityWarningAfter to
exceed InactivityTimeout; update Validate (method
SessionSupervisionConfig.Validate) to check when both c.InactivityWarningAfter >
0 and c.InactivityTimeout > 0 that c.InactivityWarningAfter <=
c.InactivityTimeout and return a clear fmt.Errorf indicating
"session.supervision.inactivity_warning_after must be <= inactivity_timeout"
(include the offending values) if the check fails; keep this check alongside the
existing range checks for c.InactivityWarningAfter and c.InactivityTimeout.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `SessionSupervisionConfig.Validate` rejects negative values but currently accepts `InactivityWarningAfter > InactivityTimeout` when both are enabled.
  - That configuration cannot emit a warning before timeout, so the fix is to reject it with a contextual validation error and add coverage.
