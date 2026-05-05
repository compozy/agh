# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 02 config lifecycle: `[tools]`, `[tools.policy]`, `[tools.hosted_mcp]`, agent `toolsets` and `deny_tools`, strict canonical grammar, overlays, examples/docs, tests, verification, tracking updates, and local commit.

## Important Decisions
- Config validation remains parse/load-time only. Runtime policy dispatch and effective decisions are left for task_03.
- Trusted external source entries use `mcp:<server_name>` and `extension:<extension_name>`. MCP entries must resolve to configured top-level/provider MCP servers; extension entries are syntax-validated because the config package does not load extensions.
- Safety bounds for this config layer: approval timeout 1-600 seconds, hosted MCP bind nonce TTL 1-300 seconds, default result budget 0-16 MiB.
- Agent `tools`/`deny_tools` accept exact canonical `ToolID`s or namespace-prefix wildcard patterns ending in `*`; `toolsets` accepts only `ToolsetID`s. No legacy `*` default is retained.

## Learnings
- `internal/tools` already owns canonical `ToolID` and `ToolsetID` validation and does not import `internal/config`, so `internal/config` can consume those validators without a cycle.
- Existing agent config defaults to `tools: ["*"]` and currently accepts ad hoc names such as `bash`; focused pre-change baseline passed with that old behavior.
- Generated API/OpenAPI/web types must move when `AgentPayload` changes; `make codegen` and `make codegen-check` covered this.
- `internal/testutil/e2e.AgentSeed` needed explicit `toolsets`/`deny_tools` support; stale `read`/`write` fixture atoms surfaced only in the race-enabled full verify path.

## Files / Surfaces
- Touched: `internal/config/config.go`, `internal/config/merge.go`, `internal/config/agent.go`, `internal/config/agent_resource.go`, `internal/config/provider.go`, `internal/config/tools.go`, `internal/config/tool_grammar.go`, config tests, `internal/api/contract/contract.go`, `internal/api/core/conversions.go`, generated OpenAPI/web types, daemon/extension/workspace clone propagation, E2E agent seed helpers, root `config.toml`, site config/agent docs, bundled agent setup example.

## Errors / Corrections
- First `make verify` failed in `internal/testutil/e2e` because `WriteAgentDef` fixtures still emitted legacy non-canonical tool names (`read`, `write`, YAML-sensitive fake names). Fixed by extending the seed helper and using canonical tool atoms.
- `openapi/agh.json` initially appeared as a huge diff after codegen; the normal verify formatter reduced it to the actual schema additions.

## Ready for Next Run
- Implementation, focused checks, codegen, self-review, task tracking, local commit `ba3aec81`, and post-commit verification are complete. Remaining work: final response only.

## Verification Evidence
- `make codegen-check` passed.
- `go test ./internal/config ./internal/daemon ./internal/workspace ./internal/extension ./internal/api/core ./internal/api/spec -count=1` passed.
- `go test ./internal/config -cover -count=1` passed with 81.8% coverage.
- `go test -race ./internal/config -count=1` passed.
- `go test -race ./internal/testutil/e2e -count=1` passed after fixture correction.
- `python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/config/tools_test.go` passed.
- `git diff --check` passed.
- Full `make verify` passed after all code changes: format, lint, typecheck, JS tests/build, Go lint, race-enabled Go tests (`DONE 6583 tests`), Go build, and package boundaries.
- Post-commit `make verify` exited 0. Tail evidence included `0 issues.`, `DONE 6583 tests in 6.078s`, and `OK: all package boundaries respected`.
