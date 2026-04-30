# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose the existing extension lifecycle as `agh__extensions` built-in tools: search/list/info plus install/update/remove/enable/disable.
- Reuse current extension manager, registry, managed install, marketplace/local-source flows, approval path, and reconciliation behavior.
- Prove trust-source denial, approval-required behavior, rollback/failure paths, and lifecycle parity with focused tests and >=80% affected-package coverage.

## Important Decisions
- Initial decision: follow Tasks 05-07 native built-in patterns and avoid any separate extension install/update/remove path.
- Added reusable managed marketplace lifecycle helpers under `internal/extension` so daemon tools call the same registry/installer/staged rollback primitives rather than ad-hoc tool-only mutation code.
- Tool install semantics infer `local` from `path` and `marketplace` from `slug`, while rejecting mixed local/marketplace inputs deterministically.
- Built-in descriptors are grouped under `agh__extensions`; read tools are read-only and lifecycle mutation tools are mutating/destructive so registry policy/approval gates run before source loaders, registry writes, or runtime reloads.

## Learnings
- Shared workflow memory says Tasks 01-07 are locally implemented/verified, so Task 08 can rely on the current `internal/tools` registry foundation and daemon native provider wiring.
- ADR-006 makes mutable extension lifecycle tool-callable by default; trust roots, raw secrets, and source policy remain the containment boundary.
- TechSpec adds no new extension manifest shape and no new top-level config keys for this task; extension lifecycle tools must project existing runtime extension points.
- Native marketplace source loading mirrors the existing extension CLI config boundary: unconfigured or unsupported marketplace registries deny deterministically with `extension_source_forbidden`.

## Files / Surfaces
- Expected working set: `internal/tools`, `internal/daemon/extensions.go`, `internal/extension/manager.go`, `internal/extension/registry.go`, `internal/extension/install_managed.go`, `internal/extension/tool_reconciliation.go`, and focused tests.
- Touched surfaces: `internal/tools/builtin_ids.go`, `internal/tools/reason.go`, `internal/tools/builtin/*`, `internal/daemon/native_tools.go`, `internal/daemon/native_extension_tools.go`, `internal/daemon/native_extension_tools_test.go`, `internal/daemon/native_extension_tools_integration_test.go`, `internal/extension/marketplace_lifecycle.go`, `internal/extension/marketplace_lifecycle_test.go`.

## Errors / Corrections
- First `make verify` attempt failed on gocritic `hugeParam` for `daemonNativeToolsDeps` and two `lll` violations; corrected by passing native deps by pointer and wrapping long declarations/calls.
- `agh-test-conventions` checker caught direct top-level assertions in new lifecycle tests; corrected by wrapping them in `t.Run("Should ...")` subtests before final verification.

## Ready for Next Run
- Implementation complete, committed, and verified locally before and after commit.
- Evidence: `go test ./internal/daemon ./internal/extension ./internal/tools ./internal/tools/builtin -count=1`; `go test -tags integration ./internal/daemon -run 'TestNativeExtensionToolsIntegrationLifecycleParity' -count=1`; `python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py` on new daemon/extension test files; `go test ./internal/extension -cover -count=1` -> 80.1%; `go test ./internal/tools/builtin -cover -run TestBuiltin -count=1` -> 93.0%; pre-commit `make verify` passed with 7070 Go tests and package boundaries respected.
- Commit: `5735b42c feat: add extension lifecycle tools`.
- Post-commit evidence: `make verify` passed with Go lint `0 issues`, `DONE 7070 tests`, and `OK: all package boundaries respected`.
- Tracking updated: `.compozy/tasks/tools-refac/task_08.md` and `_tasks.md` mark Task 08 completed.
