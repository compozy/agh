# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the task_04 plugin lifecycle in `internal/plugins/install.go` and wire `agh install` / `agh uninstall` so global plugin setup happens alongside role install.
- Cover Claude marketplace scaffolding plus CLI invocation, Codex hook/config merge and cleanup, OpenCode/Pi file copy and removal, and task-required report behavior with filesystem-safe tests under `t.TempDir()`.

## Important Decisions
- Treat the PRD, techspec, ADRs, and workflow memory as the approved design baseline; do not reopen brainstorming for this run.
- Use package-level seams in `internal/plugins` for `exec.LookPath` and external command execution so `DetectDrivers`, Claude CLI calls, and install/uninstall flows are testable without touching the real machine.
- Refactor CLI install/uninstall wiring to use injected dependencies for user-home resolution and plugin lifecycle calls, keeping CLI tests isolated from real global config roots.
- For Claude, scaffold a local marketplace under `~/.claude/plugins/marketplaces/agh-local` using the marketplace layout observed on this machine, then invoke the Claude CLI commands against that marketplace.
- For Codex, treat `codex_hooks` as a `[features]` setting in `~/.codex/config.toml` because the current CLI help maps `--enable <feature>` to `features.<name>=true`.

## Learnings
- Current-platform signals gathered during implementation prep:
  - Claude CLI is installed and manages plugins via `~/.claude/plugins`, with marketplaces stored under `~/.claude/plugins/marketplaces/<name>`.
  - Codex CLI is installed and documents `~/.codex/config.toml` as its user config root.
  - OpenCode docs currently describe the global config root as `~/.config/opencode`, with `plugins/` as a supported global subdirectory.
  - Pi is not installed on this machine, so task_04 should keep using the task/techspec path `~/.pi/agent/extensions/`.
- Claude marketplace installs use both a root `.claude-plugin/marketplace.json` manifest and per-plugin `.claude-plugin/plugin.json` files under `plugins/<plugin-name>/`.
- `plugins.Install()` now treats missing binaries as skipped drivers, while `plugins.Uninstall()` always walks the known config roots so file-based cleanup still works even if a CLI is no longer installed.
- The Codex uninstall path intentionally preserves `~/.codex/hooks.json` and removes only AGH-tagged hook commands, which keeps user hooks intact and matches the task requirement better than deleting the whole file.

## Files / Surfaces
- `internal/plugins/install.go`
- `internal/plugins/install_test.go`
- `internal/cli/install.go`
- `internal/cli/root.go`
- `internal/cli/install_test.go`
- `internal/cli/root_test.go`
- `.compozy/tasks/global-plugins/task_04.md`
- `.compozy/tasks/global-plugins/_tasks.md`
- `.codex/CONTINUITY-task-04-plugin-lifecycle.md`

## Errors / Corrections
- Existing `.codex/CONTINUITY-task-04-d2f984.md` is for an unrelated task; this run uses `.codex/CONTINUITY-task-04-plugin-lifecycle.md` instead.
- The first `make verify` run failed on an unused constant in `internal/plugins/install.go`; removed it and reran the full pipeline cleanly.
- The first install/uninstall integration assertion expected Codex `hooks.json` to be deleted; corrected the test to verify AGH hook removal while preserving the user config file.

## Ready for Next Run
- Code changes were committed locally as `89af2e0` (`feat(cli): add global plugin lifecycle`).
- Tracking and workflow-memory files remain intentionally unstaged to follow the repo rule that tracking-only files stay out of the automatic commit.
- Current unstaged task-related files are `.compozy/tasks/global-plugins/task_04.md`, `.compozy/tasks/global-plugins/_tasks.md`, and the `.compozy/tasks/global-plugins/memory/` updates.
