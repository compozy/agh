# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 is implemented in the workspace with embedded plugin assets under `internal/plugins`.
- Task 02 is implemented in the workspace with runtime env handling standardized on `AGH_*` and the `BuildHookConfig` / `HookConfig` driver API removed.
- Task 03 is implemented in the workspace with all four drivers passing prompts/config through CLI flags or env vars instead of writing workdir files.

## Shared Decisions
- `internal/plugins` is the canonical embedded asset package for the global plugin workflow. Downstream tasks should read from `plugins.Assets` or `plugins.ReadFile` rather than recreating asset content.
- The Claude hook command includes `AGH_AGENT_NAME` gating and `AGH_BIN` fallback so the embedded asset satisfies the stricter task requirements while still forwarding `PostToolUse` events to `agh hook-event`.
- Downstream runtime and driver work must use `AGH_*` env names only. Do not reintroduce `COLLAB_*` / `AGI_*` compatibility shims in this alpha line.
- `internal/kernel.AgentDriver` no longer has `BuildHookConfig`; downstream driver tasks should pass hook/runtime context through env vars and CLI arguments instead of hook-config structs.

## Shared Learnings
- The embedded asset set for task_01 is six runtime files: Claude manifest + hooks, Codex hooks + forwarder, OpenCode hook, and Pi hook.
- All four driver `buildEnv()` paths now inject `AGH_AGENT_NAME`, `AGH_SESSION_ID`, `AGH_SOCKET`, and `AGH_BIN`, and kernel/CLI runtime helpers read only `AGH_*`.
- Driver runtime transport after task_03:
  - Claude uses `--bare`, `--system-prompt`, and `--allowedTools` with no settings file.
  - Codex uses `-c developer_instructions=...` and no longer writes `.codex/AGENTS.md` or enables `codex_hooks` per session.
  - OpenCode injects the JSON config through `OPENCODE_CONFIG_CONTENT`.
  - Pi injects prompt layers through `--system-prompt` and `--append-system-prompt`.
- Task 04 confirmed the current-platform global plugin roots used by the installer:
  - Claude marketplace assets live under `~/.claude/plugins/marketplaces/<name>`.
  - Codex global config lives under `~/.codex/`.
  - OpenCode global plugins live under `~/.config/opencode/plugins/`.
  - Pi remains on the task-spec path `~/.pi/agent/extensions/`.
- Claude local marketplace installs require both a root `.claude-plugin/marketplace.json` manifest and a `plugins/<plugin>/` subtree with the embedded plugin assets; cleanup should remove the marketplace registration plus the local marketplace directory.

## Open Risks
- None currently for the global plugins workflow. Task 04 now preserves executable permissions for the Codex forwarder and covers malformed `hooks.json` recovery with tests.

## Handoffs
- Task 04 can treat the `internal/plugins` package as ready input for install/uninstall logic, and it should assume the drivers themselves no longer scaffold any per-session files in the user workdir.
