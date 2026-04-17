---
status: resolved
file: internal/automation/manager.go
line: 462
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQc,comment:PRRC_kwDOR5y4QM64dqGo
---

# Issue 021: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Resource-backed automations never get registered on startup.**

When `resourceDefinitionsEnabled()` is true, `jobs` and `triggers` stay empty here, but `loadSchedulerRegistrations` / `loadTriggerRegistrations` still run later with those slices. That boots the scheduler and trigger engine with no existing resource-defined jobs/triggers until some later write path repopulates them.

<details>
<summary>Suggested fix</summary>

```diff
 	var (
 		jobs     []Job
 		triggers []Trigger
 	)
-	if !m.resourceDefinitionsEnabled() {
-		jobs, err = m.loadEffectiveJobs(ctx, JobListQuery{})
-		if err != nil {
-			return fmt.Errorf("automation: load effective jobs: %w", err)
-		}
-		triggers, err = m.loadEffectiveTriggers(ctx, TriggerListQuery{})
-		if err != nil {
-			return fmt.Errorf("automation: load effective triggers: %w", err)
-		}
+	jobs, err = m.loadEffectiveJobs(ctx, JobListQuery{})
+	if err != nil {
+		return fmt.Errorf("automation: load effective jobs: %w", err)
+	}
+	triggers, err = m.loadEffectiveTriggers(ctx, TriggerListQuery{})
+	if err != nil {
+		return fmt.Errorf("automation: load effective triggers: %w", err)
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	var (
		jobs     []Job
		triggers []Trigger
	)
	jobs, err = m.loadEffectiveJobs(ctx, JobListQuery{})
	if err != nil {
		return fmt.Errorf("automation: load effective jobs: %w", err)
	}
	triggers, err = m.loadEffectiveTriggers(ctx, TriggerListQuery{})
	if err != nil {
		return fmt.Errorf("automation: load effective triggers: %w", err)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/manager.go` around lines 449 - 462, The startup guard
around resourceDefinitionsEnabled() causes jobs and triggers to remain empty
when resource-backed automations are enabled, so loadSchedulerRegistrations and
loadTriggerRegistrations start with nothing; remove or change the conditional so
that loadEffectiveJobs and loadEffectiveTriggers are always invoked (or invoke
the appropriate resource-backed loaders) regardless of
resourceDefinitionsEnabled(), ensuring the jobs and triggers slices are
populated before calling loadSchedulerRegistrations/loadTriggerRegistrations
(refer to resourceDefinitionsEnabled(), loadEffectiveJobs,
loadEffectiveTriggers, loadSchedulerRegistrations, and
loadTriggerRegistrations).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `Manager.Start` still skips `loadEffectiveJobs` and `loadEffectiveTriggers` when resource definitions are enabled, but it always bootstraps the scheduler and trigger engine from those slices immediately afterward. That leaves resource-backed jobs and triggers unregistered until a later write path repopulates runtime state. The fix is to always load effective jobs and triggers before startup registration. Validation needs a minimal out-of-scope test update in `internal/automation/resource_test.go` because the scoped batch does not include an automation startup test file.
