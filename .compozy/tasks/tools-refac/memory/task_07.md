# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 07 "Automation Tool Family": expose automation jobs, triggers, manual trigger operations, and run inspection/history through canonical built-in tools while reusing existing automation manager, validators, and persistence. Mutating operations must require approval and return deterministic policy/validation denials.

## Important Decisions
- Reuse the existing tools-refac native built-in pattern (`internal/tools/builtin` descriptors plus `internal/daemon` native handlers) instead of creating a parallel automation tool API.
- Keep automation storage, scheduling, dispatch, and validation authority in `internal/automation`; tools are an adapter over those paths.
- Register a dedicated `agh__automation` native toolset. Include TechSpec-listed jobs/triggers/runs tools and explicit job/trigger enable-disable tools because Task 07 subtasks require them.
- Treat raw webhook secret material as operator-only: any `webhook_secret` field supplied through automation tools is denied before calling the automation manager.

## Learnings
- Shared workflow memory confirms the tool registry, policy resolver, approval bridge, and prior read/mutable tool families are already implemented on this branch.
- ADR-006 and the TechSpec require mutable automation management to be tool-callable by default, with operator-only reserved for trust-root, raw-secret, and human-interactive boundaries.
- Current branch has no `agh__automation` descriptors or daemon native handlers yet; authoritative automation read/write paths already exist on `core.AutomationManager`.
- Automation tool implementation now exists as a native built-in toolset and reuses API DTO conversion/patch helpers plus `core.AutomationManager`; no storage, scheduler, or dispatcher behavior was forked.
- Focused unit tests cover descriptor/toolset registration, approval gating, deterministic validation/source/secret denials, native manager routing, config-backed enable-only updates, manager error mapping, and run-history query filters.
- Focused integration test uses real `automation.Manager` + `globaldb` to prove create/update/enable/disable/trigger/history/delete tool calls mutate and inspect the same persisted state as existing manager paths.
- Full pre-commit `make verify` passed after the `ToolsetCatalog` `funlen` correction with Go lint `0 issues`, `DONE 7059 tests`, and package boundaries respected.
- Self-review compared native automation handlers against `internal/api/core/automation.go`; create/update/delete/trigger/history semantics now intentionally mirror the public automation management path, except raw webhook secret input is rejected at the tool boundary as an operator-only raw-secret boundary.
- Created local commit `06880bab feat: add automation management tools` containing only Task 07 code/test changes.
- Post-commit `make verify` passed with Go lint `0 issues`, `DONE 7059 tests`, and package boundaries respected.

## Files / Surfaces
- Touched implementation surfaces: `internal/tools/builtin_ids.go`, `internal/tools/reason.go`, `internal/tools/builtin/{automation.go,descriptors.go,toolsets.go}`, `internal/daemon/{native_tools.go,native_automation_tools.go}`, and `internal/api/core/automation.go`.
- Touched test surfaces: `internal/tools/builtin/builtin_test.go`, `internal/daemon/{native_tools_test.go,native_automation_tools_test.go,native_automation_tools_integration_test.go}`.
- Scope stayed in tool IDs/descriptors, native daemon handlers, shared API automation helper wrappers, and tests; no automation storage or scheduler fork.

## Errors / Corrections
- Corrected automation history input decoding to embed run-query fields at the top level, matching the public tool schema.
- Corrected trigger test fixtures to use valid trigger filter/template paths (`data.*`, `.Kind`) from existing automation validators.
- First full `make verify` failed on `funlen` after automation pushed `ToolsetCatalog` over 80 lines; fixed by extracting the built-in toolset list to `builtinToolsets` without changing catalog semantics.

## Ready for Next Run
- Grounding completed before code edits: workflow memory, Task 07, `_tasks.md`, `_techspec.md` automation/mutable-policy/agent-manageability sections, ADR-001 through ADR-006, root/internal guidance, and Go/test skills were read.
- Focused checks passed: `go test ./internal/tools ./internal/tools/builtin ./internal/api/core ./internal/daemon -run 'Automation|BuiltinNativeDescriptors|BuiltinToolsetCatalog|DaemonNativeTools'`; `go test -tags integration ./internal/daemon -run TestDaemonNativeAutomationToolsIntegrationLifecycleParity`.
- Focused coverage passed for the new native handler file: `native_automation_tools.go` 80.5% (256/318 statements) from `/tmp/agh-automation-tools-daemon.cover`; whole `internal/daemon` package reports 13.2% because the package is much larger than this task scope.
- Tracking updated after verification/self-review: `task_07.md` status and subtask/test checkboxes are complete, and `_tasks.md` marks Task 07 complete.
- Final state: local commit `06880bab` exists; pre-commit and post-commit `make verify` both passed.
