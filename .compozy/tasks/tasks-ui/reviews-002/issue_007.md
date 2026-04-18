---
status: resolved
file: internal/observe/observer.go
line: 242
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM576AUh,comment:PRRC_kwDOR5y4QM65ChGw
---

# Issue 007: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Make repeated dashboard-config options composable.**

Line 219 rebuilds the config from defaults every time. If `WithTaskDashboardConfig(...)` is applied more than once, a later partial override silently clears earlier overrides instead of layering on top of them.



<details>
<summary>♻️ Proposed fix</summary>

```diff
 func WithTaskDashboardConfig(cfg TaskDashboardConfig) Option {
 	return func(observer *Observer) {
-		observer.taskDashboardConfig = normalizeTaskDashboardConfig(cfg)
+		if cfg.ActiveRunLimit > 0 {
+			observer.taskDashboardConfig.activeRunLimit = cfg.ActiveRunLimit
+		}
+		if cfg.BacklogWarnAfter > 0 {
+			observer.taskDashboardConfig.backlogWarnAfter = cfg.BacklogWarnAfter
+		}
+		if cfg.StaleAfter > 0 {
+			observer.taskDashboardConfig.staleAfter = cfg.StaleAfter
+		}
 	}
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/observer.go` around lines 217 - 242, The current
WithTaskDashboardConfig replaces the whole taskDashboardConfig with defaults
then the new partial cfg, so applying it multiple times loses previous
overrides; change the merge behavior so new options layer on the existing
config: either update normalizeTaskDashboardConfig to accept a base
taskDashboardConfig and apply non-zero fields from TaskDashboardConfig onto that
base (using observer.taskDashboardConfig as the base) or modify
WithTaskDashboardConfig to start from observer.taskDashboardConfig (or
defaultTaskDashboardConfig() if zero) and then overlay cfg's non-zero fields
before assigning back to observer.taskDashboardConfig; refer to
WithTaskDashboardConfig, normalizeTaskDashboardConfig,
defaultTaskDashboardConfig and taskDashboardConfig when making this change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `WithTaskDashboardConfig` currently normalizes from defaults every time and assigns the whole config. If the option is applied more than once, later partial overrides discard earlier non-default settings.
- Root cause analysis: The option function rebuilds from `defaultTaskDashboardConfig()` instead of layering onto the observer’s existing normalized config.
- Intended fix: Merge partial dashboard config overrides onto the current observer config and add coverage that exercises multiple `WithTaskDashboardConfig(...)` applications.
- Resolution: Changed `WithTaskDashboardConfig(...)` to merge onto the observer’s existing normalized dashboard config and added coverage for layered partial overrides.
- Verification:
  - `go test ./internal/extension ./internal/observe`
  - `go test -tags integration ./internal/observe -run 'TestObserveTaskDashboard|TestObserveHealthReflectsRecoveryAndForcedStopOutcomes|TestObserveTaskLifecycleSummaryAndMetrics'`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
