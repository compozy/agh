---
status: resolved
file: internal/cli/lifecycle.go
line: 163
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLiS,comment:PRRC_kwDOR5y4QM67SmDV
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Managed `uninstall` is still gated by local validation and runtime loading.**

Lines 143-149 run `--purge` validation and `loadRuntimeContext(deps)` before the managed check. That means `AGH_MANAGED=... agh uninstall --purge` errors on `--force`, and any runtime-context failure prevents the deferred “use your package manager” response.



<details>
<summary>Suggested fix</summary>

```diff
		RunE: func(cmd *cobra.Command, _ []string) error {
+			state := detectManagedState(deps)
+			if state.Managed {
+				record := lifecycleRecord{
+					Command: "uninstall",
+					Managed: state.Managed,
+					Manager: state.Manager,
+					Status:  lifecycleStatusDeferred,
+					Message: "AGH is managed by an external package manager; no local uninstall changes were made.",
+					Recommendation: managedRecommendation(state.Manager, "uninstall AGH"),
+				}
+				return writeCommandOutput(cmd, lifecycleBundle("Uninstall", record))
+			}
+
			if purge && !force {
				return errors.New("cli: --purge requires --force to remove AGH home data")
			}

			runtime, err := loadRuntimeContext(deps)
			if err != nil {
				return err
			}

-			state := detectManagedState(deps)
			record := lifecycleRecord{
				Command: "uninstall",
				HomeDir: runtime.HomePaths.HomeDir,
				Managed: state.Managed,
				Manager: state.Manager,
			}
-			if state.Managed {
-				record.Status = lifecycleStatusDeferred
-				record.Message = "AGH is managed by an external package manager; no local uninstall changes were made."
-				record.Recommendation = managedRecommendation(state.Manager, "uninstall AGH")
-				return writeCommandOutput(cmd, lifecycleBundle("Uninstall", record))
-			}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/lifecycle.go` around lines 143 - 163, The managed-uninstall flow
currently validates --purge/--force and calls loadRuntimeContext(deps) before
checking detectManagedState, causing managed installs to error on local
validations and runtime failures; change the logic in the uninstall handler to
call detectManagedState() first and if state.Managed is true immediately
construct the lifecycleRecord (using lifecycleRecord, lifecycleStatusDeferred,
managedRecommendation(state.Manager, "uninstall AGH")) and return
writeCommandOutput(cmd, lifecycleBundle("Uninstall", record)) without performing
the --purge validation or calling loadRuntimeContext(deps); only perform the
purge/force check and loadRuntimeContext when state.Managed is false.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `newUninstallCommand` validates `--purge/--force` and loads runtime context before checking managed install state, so managed installs can fail local validation or runtime loading instead of returning the deferred package-manager response.
- Fix approach: detect managed state at the start of the uninstall handler and return the managed advisory output before purge validation or runtime context loading; keep local validation for unmanaged installs only.
