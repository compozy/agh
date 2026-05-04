---
status: resolved
file: internal/cli/lifecycle.go
line: 121
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLiQ,comment:PRRC_kwDOR5y4QM67SmDT
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Short-circuit managed `update` before resolving local home.**

Line 106 makes `update` depend on `deps.resolveHome()` even though the managed path is purely advisory. A managed install with a broken or missing local home will fail here instead of returning the package-manager guidance.



<details>
<summary>Suggested fix</summary>

```diff
		RunE: func(cmd *cobra.Command, _ []string) error {
-			homePaths, err := deps.resolveHome()
-			if err != nil {
-				return err
-			}
			state := detectManagedState(deps)
			record := lifecycleRecord{
				Command: "update",
-				HomeDir: homePaths.HomeDir,
				Managed: state.Managed,
				Manager: state.Manager,
			}
			if state.Managed {
				record.Status = lifecycleStatusDeferred
				record.Message = "AGH is managed by an external package manager; no local update was performed."
				record.Recommendation = managedRecommendation(state.Manager, "update AGH")
				return writeCommandOutput(cmd, lifecycleBundle("Update", record))
			}
+			homePaths, err := deps.resolveHome()
+			if err != nil {
+				return err
+			}
+			record.HomeDir = homePaths.HomeDir

			record.Status = lifecycleStatusManual
			record.Message = "No in-place updater is configured for this unmanaged AGH binary; no files were changed."
			record.Recommendation = "Install a newer release archive, rerun `go install`, or rebuild from source."
			return writeCommandOutput(cmd, lifecycleBundle("Update", record))
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/lifecycle.go` around lines 106 - 121, The update command
currently calls deps.resolveHome() before checking managed state, causing
managed installs to fail if local home is missing; move the
detectManagedState(deps) call and the managed-install short-circuit (creating
lifecycleRecord with Command "update", Managed/state.Manager, setting Status to
lifecycleStatusDeferred, Message and Recommendation via managedRecommendation)
to run before calling deps.resolveHome(), and only call deps.resolveHome() and
set HomeDir on the lifecycleRecord when state.Managed is false; ensure
writeCommandOutput(lifecycleBundle("Update", record)) is still returned for the
managed path.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `newUpdateCommand` calls `deps.resolveHome()` before checking `detectManagedState`, so a managed install can fail on local home resolution even though the managed update path only needs to return package-manager guidance.
- Fix approach: detect managed state first and short-circuit managed update output before resolving local home paths; keep home resolution only for unmanaged binaries.
