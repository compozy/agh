# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Extend `internal/config` and agent-definition loading so config layers and agent defs can declare hooks, expose combined `[]hooks.HookDecl` for the registry, enforce task_02 validation/defaults, and satisfy task_08 tests plus full repo verification.

## Important Decisions
- Added package-level `HookDeclarations(cfg Config, agents []AgentDef) ([]hooks.HookDecl, error)` as the registry-facing export.
- Config hook declarations are parsed into `Config.Hooks.Declarations` and merged across precedence layers by declaration name, with later layers replacing matching names.
- Agent-definition hooks are always scoped to the defining agent name via `matcher.agent_name`; mismatched explicit values fail parsing.
- Agent frontmatter decoding now supports strict YAML first and strict TOML fallback so hook declarations work in both metadata formats.
- `session.Notifier.OnAgentEvent` now accepts `any`, with `internal/observe` performing ACP downcasting, to avoid a new import cycle caused by config-driven hook declarations.

## Learnings
- `internal/hooks.ValidateHookDecl` is enough for load-time validation, while `NormalizeHookDecl` is the right path for applying default priority and executor kind before registry consumption.
- The existing frontmatter splitter is format-agnostic; YAML/TOML support can be layered entirely in the decode callback without changing `internal/frontmatter`.

## Files / Surfaces
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/config/agent.go`
- `internal/config/bootstrap.go`
- `internal/config/hooks.go`
- `internal/config/hooks_test.go`
- `internal/workspace/clone.go`
- `internal/session/interfaces.go`
- `internal/hooks/agent_event.go`
- `internal/observe/observer.go`
- `internal/daemon/notifier.go`
- `internal/daemon/notifier_test.go`
- `internal/daemon/daemon_test.go`
- `internal/session/manager_test.go`
- `internal/cli/cli_integration_test.go`
- `internal/api/httpapi/httpapi_integration_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`
- `internal/hooks/hooks_test.go`

## Errors / Corrections
- Initial implementation covered YAML agent frontmatter only; corrected by adding strict TOML fallback to satisfy the task requirement for YAML/TOML agent definitions.
- Fixed a compile error in the TOML unknown-field path by using `toml.Key.String()` instead of passing a single `toml.Key` into the config overlay helper that expects a slice.

## Ready for Next Run
- Verification complete after the final code change:
  - `go test ./internal/config -count=1`
  - `go test -cover ./internal/config -count=1` (`82.5%`)
  - `make verify` (exit `0`)
- Task tracking is updated locally and the code-only commit is `ee38729` (`feat: add config hook declarations`).
- Committed `HEAD` was re-verified with `make verify` (exit `0`).
