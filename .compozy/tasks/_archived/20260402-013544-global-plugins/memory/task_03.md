# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Remove all per-session workdir writes from the Claude, Codex, OpenCode, and Pi drivers.
- Replace file transports with ADR-005 transports: Claude `--bare`/CLI-only, Codex `-c developer_instructions=...`, OpenCode `OPENCODE_CONFIG_CONTENT`, Pi `--system-prompt` plus `--append-system-prompt`.
- Keep `ParseHookEvent()` unchanged and preserve non-file-write runtime behavior.

## Important Decisions
- Use primary-source CLI/help verification before editing:
  - Claude help confirms `--bare`, `--system-prompt`, `--allowedTools`
  - Codex help plus OpenAI Codex config reference confirm `-c/--config` and `developer_instructions`
  - Pi upstream README/source confirm `--system-prompt` and `--append-system-prompt`
  - OpenCode SDK source confirms `OPENCODE_CONFIG_CONTENT`
- Keep scope limited to the four drivers and their tests; do not widen into install/uninstall or unrelated OpenCode CLI mode cleanup.
- Encode Codex prompt transport with a TOML-compatible `developer_instructions` config override so multiline prompts survive as a single `-c` argument.

## Learnings
- Current code still writes `.codex/AGENTS.md`, `opencode.json`, `.pi/SYSTEM.md`, and `.pi/AGENTS.md`.
- Claude is already using `--bare` and no longer emits `--settings`; its task_03 work is mostly test/cleanup confirmation.
- `config_path`, `agents_path`, and `system_path` metadata fields are only produced by the current write paths and are not referenced elsewhere in production code.
- After the refactor, the four driver packages exceed the required coverage target:
  - Claude 85.7%
  - Codex 85.8%
  - OpenCode 82.1%
  - Pi 86.6%
- `make verify` passes cleanly with the driver changes in place.
- Grep confirms there are no `os.WriteFile`, `os.MkdirAll`, or `BuildHookConfig` references left in the four driver source packages.
- Local commit created: `e091d08` (`Refactor drivers to avoid workdir writes`).

## Files / Surfaces
- `internal/drivers/claude/claude.go`
- `internal/drivers/codex/codex.go`
- `internal/drivers/opencode/opencode.go`
- `internal/drivers/pi/pi.go`
- `internal/drivers/claude/claude_test.go`
- `internal/drivers/codex/codex_test.go`
- `internal/drivers/opencode/opencode_test.go`
- `internal/drivers/pi/pi_test.go`

## Errors / Corrections
- Initial targeted `go test ./internal/drivers/...` run failed because `claude_test.go` still imported `filepath` and `codex.go` still needed `filepath` for `resolveHookCommandPath`; fixed the imports and reran successfully.

## Ready for Next Run
- Task is complete. If follow-up work appears, it should start from task_04 or from any future CLI transport compatibility regressions rather than reopening the removed workdir write paths.
