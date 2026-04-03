# TechSpec: Global Plugins and Zero Workdir Pollution

## Executive Summary

AGH drivers generate 7 files in the user's working directory every time an agent starts — hook configurations, system prompts, and runtime settings. These files are never cleaned up and pollute the user's project. This techspec eliminates all per-session file writes by migrating hooks to permanent global plugins/extensions and passing runtime config via CLI flags and environment variables. Additionally, all environment variables are standardized from the dual `COLLAB_*`/`AGI_*` prefixes to a unified `AGH_*` prefix.

Key architectural decisions: global plugins over per-session files (ADR-001), hybrid plugin strategy (ADR-002), `AGH_*` env var standardization (ADR-003), `BuildHookConfig` removal from driver interface (ADR-004), and zero workdir pollution via CLI flags (ADR-005).

## System Architecture

### Component Overview

```
┌──────────────────────────────────────────────────────────────┐
│                    agh install (one-time)                      │
│                                                                │
│  Reads embedded plugin assets from internal/plugins/           │
│  │                                                             │
│  ├─ Claude: creates local marketplace + plugin install         │
│  │   → ~/.claude/plugins/agh/                                  │
│  │                                                             │
│  ├─ Codex: merge hooks.json + copy forwarder script            │
│  │   → ~/.codex/hooks.json + ~/.codex/hooks/agh-forwarder.sh   │
│  │   → ~/.codex/config.toml (ensure codex_hooks=true)          │
│  │                                                             │
│  ├─ OpenCode: copy plugin file                                 │
│  │   → ~/.config/opencode/plugins/agh-hook.ts                  │
│  │                                                             │
│  └─ Pi: copy extension file                                    │
│      → ~/.pi/agent/extensions/agh-hook.ts                      │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                  Agent Start (every session)                   │
│                                                                │
│  driver.Start(ctx, startOpts)                                  │
│  │                                                             │
│  ├─ buildEnv() sets AGH_* env vars                             │
│  │   AGH_AGENT_NAME, AGH_SESSION_ID, AGH_SOCKET, AGH_BIN      │
│  │                                                             │
│  ├─ buildCommand() passes config via flags/env:                │
│  │   Claude: --bare --system-prompt --allowedTools             │
│  │   Codex:  -c 'developer_instructions=...'                   │
│  │   OpenCode: OPENCODE_CONFIG_CONTENT env var                 │
│  │   Pi:     --system-prompt --append-system-prompt --tools    │
│  │                                                             │
│  └─ NO FILES WRITTEN TO WORKDIR                                │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                  Hook Event Flow (runtime)                     │
│                                                                │
│  Agent CLI (claude/codex/pi/opencode)                          │
│  │                                                             │
│  ├─ Global plugin detects AGH_AGENT_NAME in env                │
│  ├─ Tool event fires (PostToolUse, tool_execution_end, etc.)   │
│  ├─ Plugin calls: agh hook-event --agent $AGH_AGENT_NAME       │
│  │   (reads AGH_SESSION_ID, AGH_SOCKET from env)               │
│  │                                                             │
│  └─ HookRouter receives event via NATS → routes to master      │
│                                                                │
│  OpenCode: unchanged (SSE stream, no plugin needed for hooks)  │
└──────────────────────────────────────────────────────────────┘
```

### Data Flow

1. `agh install` extracts embedded plugin files and installs them in each CLI's global directory
2. At agent spawn, `buildEnv()` sets `AGH_*` env vars on the child process
3. `buildCommand()` passes system prompt and tools via CLI flags or env vars (no files)
4. The agent CLI starts and auto-loads the global plugin/extension
5. Plugin checks `AGH_AGENT_NAME` — if absent, no-ops silently
6. On tool events, plugin calls `agh hook-event --agent $AGH_AGENT_NAME` with payload on stdin
7. `agh hook-event` reads `AGH_SESSION_ID` and `AGH_SOCKET` from env, sends to kernel via UDS
8. `HookRouter` calls `driver.ParseHookEvent()` to normalize, then routes to master agent

## Implementation Design

### 1. Embedded Plugin Assets

New package `internal/plugins/` with `go:embed` assets:

```
internal/plugins/
  embed.go                  ← //go:embed all:claude all:codex all:opencode all:pi
  install.go                ← Install(), Uninstall(), DetectDrivers()
  claude/
    .claude-plugin/
      plugin.json           ← manifest with name, description, version
    hooks/
      hooks.json            ← PostToolUse → agh hook-event --agent $AGH_AGENT_NAME
  codex/
    hooks.json              ← PostToolUse → ~/.codex/hooks/agh-forwarder.sh
    agh-forwarder.sh        ← reads stdin, calls agh hook-event
  opencode/
    agh-hook.ts             ← tool.execute.before/after → agh hook-event
  pi/
    agh-hook.ts             ← pi.on(tool_execution_start/end) → agh hook-event
```

```go
package plugins

import "embed"

//go:embed all:claude all:codex all:opencode all:pi
var Assets embed.FS
```

### 2. Plugin Content

#### Claude — `claude/.claude-plugin/plugin.json`

```json
{
  "name": "agh",
  "description": "AGH kernel hook forwarder — forwards tool events to AGH daemon",
  "version": "1.0.0"
}
```

#### Claude — `claude/hooks/hooks.json`

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "agh hook-event --agent $AGH_AGENT_NAME"
          }
        ]
      }
    ]
  }
}
```

#### Codex — `codex/hooks.json`

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "~/.codex/hooks/agh-forwarder.sh",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

#### Codex — `codex/agh-forwarder.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail
INPUT=$(cat)
AGENT="${AGH_AGENT_NAME:-}"
[ -z "$AGENT" ] && { echo '{}'; exit 0; }
echo "$INPUT" | "${AGH_BIN:-agh}" hook-event --agent "$AGENT" 2>/dev/null || true
echo '{}'
exit 0
```

#### OpenCode — `opencode/agh-hook.ts`

```typescript
import type { Plugin } from "@opencode-ai/plugin"

export const AghHook: Plugin = async ({ $ }) => {
  const agent = process.env.AGH_AGENT_NAME
  if (!agent) return {}

  const aghBin = process.env.AGH_BIN ?? "agh"

  const forward = async (payload: Record<string, unknown>) => {
    try {
      const json = JSON.stringify(payload)
      await $`echo ${json} | ${aghBin} hook-event --agent ${agent}`.quiet()
    } catch {}
  }

  return {
    "tool.execute.before": async (input, output) => {
      await forward({
        agent_name: agent,
        type: "tool_call",
        tool_name: input.tool,
        tool_input: output.args,
        timestamp: new Date().toISOString(),
      })
    },
    "tool.execute.after": async (input) => {
      await forward({
        agent_name: agent,
        type: "tool_result",
        tool_name: input.tool,
        tool_input: input.args,
        timestamp: new Date().toISOString(),
      })
    },
  }
}
```

#### Pi — `pi/agh-hook.ts`

```typescript
import { execSync } from "node:child_process"

export default function (pi: any) {
  const agent = process.env.AGH_AGENT_NAME
  if (!agent) return

  const aghBin = process.env.AGH_BIN ?? "agh"

  const forward = (event: unknown) => {
    try {
      execSync(`${aghBin} hook-event --agent ${agent}`, {
        input: JSON.stringify(event),
        timeout: 5000,
        stdio: ["pipe", "pipe", "pipe"],
      })
    } catch {}
  }

  pi.on("tool_execution_start", async (e: unknown) => forward(e))
  pi.on("tool_execution_end", async (e: unknown) => forward(e))
}
```

### 3. Install/Uninstall Logic

`internal/plugins/install.go`:

```go
func Install(homeDir string) (*InstallReport, error)
func Uninstall(homeDir string) (*UninstallReport, error)
func DetectDrivers() []string
```

**`DetectDrivers()`**: Uses `exec.LookPath` to check which CLIs are available (`claude`, `codex`, `opencode`, `pi`). Returns list of detected driver names.

**`Install()`**:
1. Calls `DetectDrivers()` to find available CLIs
2. For each detected driver:
   - **Claude**: Creates temp dir with plugin structure, runs `claude plugin install` from local marketplace, scope user
   - **Codex**: Reads existing `~/.codex/hooks.json`, merges AGH hooks (tagged with `"_agh": true` marker), writes back. Copies `agh-forwarder.sh` to `~/.codex/hooks/`. Ensures `codex_hooks = true` in `~/.codex/config.toml`
   - **OpenCode**: Copies `agh-hook.ts` to `~/.config/opencode/plugins/`
   - **Pi**: Copies `agh-hook.ts` to `~/.pi/agent/extensions/`
3. Returns report with installed/skipped drivers and any errors

**`Uninstall()`**:
1. **Claude**: Runs `claude plugin uninstall agh --scope user`
2. **Codex**: Reads `~/.codex/hooks.json`, removes entries tagged `"_agh": true`, writes back. Removes `~/.codex/hooks/agh-forwarder.sh`
3. **OpenCode**: Removes `~/.config/opencode/plugins/agh-hook.ts`
4. **Pi**: Removes `~/.pi/agent/extensions/agh-hook.ts`

### 4. Driver Changes

#### AgentDriver Interface

Remove from `internal/kernel/types.go`:

```go
// REMOVE:
BuildHookConfig(agentName string, hookEndpoint string) (*HookConfig, error)

// REMOVE entire struct:
type HookConfig struct { ... }
```

Updated interface (6 methods):

```go
type AgentDriver interface {
    Name() string
    Start(ctx context.Context, opts StartOpts) (*AgentProcess, error)
    SendMessage(ctx context.Context, proc *AgentProcess, msg string) error
    Stop(ctx context.Context, proc *AgentProcess) error
    ParseHookEvent(rawPayload []byte) (*HookEvent, error)
    HealthCheck(ctx context.Context, proc *AgentProcess) (AgentHealth, error)
    DetectReady(ctx context.Context, proc *AgentProcess) error
}
```

#### Claude Driver

`internal/drivers/claude/claude.go`:

**`Start()`**: Remove hook config generation, directory creation, and file writes (lines 187-202). Just build command and start process.

**`buildCommand()`**: Replace `--settings {path}` with `--bare`. Keep `--system-prompt`, `--allowedTools`, `--model`, `--name`.

```go
args := []string{
    "--bare",
    "--dangerously-skip-permissions",
    "--model", model,
    "--name", opts.Name,
    "--system-prompt", opts.SystemPrompt,
}
if len(tools) > 0 {
    args = append(args, "--allowedTools", strings.Join(tools, ","))
}
```

**`buildEnv()`**: Replace `AGI_AGENT_NAME` with `AGH_AGENT_NAME`. Add `AGH_SESSION_ID`, `AGH_SOCKET`, `AGH_BIN`.

#### Codex Driver

`internal/drivers/codex/codex.go`:

**`Start()`**: Remove AGENTS.md write (lines 178-186) and hooks.json write (lines 188-196). Just build command and start process.

**`buildCommand()`**: Remove `--enable codex_hooks`. Add `-c` flag for system prompt.

```go
args := []string{
    "--yolo",
    "-m", model,
    "-C", opts.WorkDir,
    "--sandbox", sandboxMode,
    "-c", fmt.Sprintf("developer_instructions=%q", opts.SystemPrompt),
}
```

**`buildEnv()`**: Same env var migration as Claude.

#### OpenCode Driver

`internal/drivers/opencode/opencode.go`:

**`Start()`**: Remove `opencode.json` write (lines 264-272). Pass config via env var instead.

**`buildEnv()`**: Add `OPENCODE_CONFIG_CONTENT` with the JSON that was previously written to file. Migrate to `AGH_*` prefix.

```go
func buildEnv(opts kernel.StartOpts) []string {
    configJSON := buildConfigJSON(opts) // same as buildConfig() but returns string
    merged := inheritEnv(opts.EnvVars)
    merged["AGH_AGENT_NAME"] = opts.Name
    merged["AGH_SESSION_ID"] = opts.SessionID
    merged["AGH_SOCKET"] = opts.SocketPath
    merged["AGH_BIN"] = aghBinaryPath()
    merged["OPENCODE_CONFIG_CONTENT"] = configJSON
    return envMapToSlice(merged)
}
```

**`buildCommand()`**: Unchanged (already uses `--model`, `--dir`, `--agent`, etc.).

#### Pi Driver

`internal/drivers/pi/pi.go`:

**`Start()`**: Remove SYSTEM.md write (lines 185-191), AGENTS.md write (lines 193-197), and agh-hook.ts write (lines 199-210). Just build command and start process.

**`buildCommand()`**: Add `--system-prompt` and `--append-system-prompt`.

```go
args := []string{
    "--cwd", opts.WorkDir,
    "--model", model,
    "--system-prompt", opts.SystemPrompt,
    "--append-system-prompt", buildAdditionalContext(opts),
}
if len(tools) > 0 {
    args = append(args, "--tools", strings.Join(tools, ","))
}
```

**`buildEnv()`**: Same env var migration as Claude.

### 5. Environment Variable Migration

Global rename across all files:

| File(s) | Change |
|---------|--------|
| `internal/drivers/claude/claude.go` | `buildEnv()`: `AGI_AGENT_NAME` → `AGH_AGENT_NAME`, add `AGH_SESSION_ID`, `AGH_SOCKET`, `AGH_BIN` |
| `internal/drivers/codex/codex.go` | Same |
| `internal/drivers/opencode/opencode.go` | Same |
| `internal/drivers/pi/pi.go` | Same |
| `internal/cli/hooks.go` | Read `AGH_SESSION_ID`, `AGH_SOCKET`, `AGH_AGENT` instead of dual `COLLAB_*`/`AGI_*` |
| `internal/cli/*_test.go` | Update test env var references |
| `internal/drivers/*_test.go` | Update test env var references |
| `internal/kernel/session_manager.go` | If it propagates env vars, update prefixes |

### 6. CLI Changes

#### `agh install` (updated)

`internal/cli/install.go`:

```go
func installCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "install",
        Short: "Install AGH roles and driver plugins",
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. Install roles (existing behavior)
            if err := roles.Install(homeDir); err != nil {
                return fmt.Errorf("install roles: %w", err)
            }
            // 2. Install driver plugins (new)
            report, err := plugins.Install(homeDir)
            if err != nil {
                return fmt.Errorf("install plugins: %w", err)
            }
            // 3. Print report
            printInstallReport(report)
            return nil
        },
    }
}
```

#### `agh uninstall` (new)

```go
func uninstallCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "uninstall",
        Short: "Remove AGH driver plugins from global config",
        RunE: func(cmd *cobra.Command, args []string) error {
            report, err := plugins.Uninstall(homeDir)
            if err != nil {
                return fmt.Errorf("uninstall plugins: %w", err)
            }
            printUninstallReport(report)
            return nil
        },
    }
}
```

### 7. HookRouter Changes

`internal/kernel/hooks.go`:

The `HookRouter` itself does not change. It receives events through NATS regardless of how the hook command was triggered. The only change is that `Start()` no longer calls `BuildHookConfig()` — the router's subscription and routing logic remains identical.

Remove any code in `session_manager.go` or `api.go` that calls `BuildHookConfig()` and writes the result to disk.

## Files to Create

| File | Purpose |
|------|---------|
| `internal/plugins/embed.go` | `go:embed` declarations for plugin assets |
| `internal/plugins/install.go` | `Install()`, `Uninstall()`, `DetectDrivers()` logic |
| `internal/plugins/install_test.go` | Tests for install/uninstall/detect |
| `internal/plugins/claude/.claude-plugin/plugin.json` | Claude plugin manifest |
| `internal/plugins/claude/hooks/hooks.json` | Claude hook configuration |
| `internal/plugins/codex/hooks.json` | Codex hook configuration |
| `internal/plugins/codex/agh-forwarder.sh` | Codex hook forwarder script |
| `internal/plugins/opencode/agh-hook.ts` | OpenCode plugin |
| `internal/plugins/pi/agh-hook.ts` | Pi extension |

## Files to Modify

| File | Change |
|------|--------|
| `internal/kernel/types.go` | Remove `BuildHookConfig` from `AgentDriver`, remove `HookConfig` struct |
| `internal/drivers/claude/claude.go` | Remove file writes from `Start()`, add `--bare` to `buildCommand()`, migrate env vars |
| `internal/drivers/codex/codex.go` | Remove file writes from `Start()`, add `-c developer_instructions` to `buildCommand()`, migrate env vars |
| `internal/drivers/opencode/opencode.go` | Remove file write from `Start()`, add `OPENCODE_CONFIG_CONTENT` to `buildEnv()`, migrate env vars |
| `internal/drivers/pi/pi.go` | Remove file writes from `Start()`, add `--system-prompt` to `buildCommand()`, migrate env vars |
| `internal/cli/hooks.go` | Read `AGH_*` env vars instead of `COLLAB_*`/`AGI_*` |
| `internal/cli/install.go` | Add plugin installation to existing install command |
| `internal/cli/root.go` | Register `agh uninstall` command |
| `internal/kernel/session_manager.go` | Remove `BuildHookConfig` calls, migrate env vars if applicable |
| `internal/kernel/api.go` | Remove `BuildHookConfig` calls if present |
| All `*_test.go` files for above | Update to match new behavior and env var names |

## Architecture Decision Records

- [ADR-001: Global Plugins Over Per-Session File Generation](adrs/adr-001.md) — Replace per-session hook file generation with permanent global plugins installed once per machine.
- [ADR-002: Hybrid Plugin Strategy](adrs/adr-002.md) — Use formal plugin isolation where CLI supports it (Claude), file merge where it does not (Codex), and standalone files where natural (OpenCode, Pi).
- [ADR-003: AGH_* Environment Variable Prefix Standardization](adrs/adr-003.md) — Migrate all environment variables from dual `COLLAB_*`/`AGI_*` prefixes to unified `AGH_*`.
- [ADR-004: Remove BuildHookConfig from AgentDriver Interface](adrs/adr-004.md) — Remove the method and `HookConfig` struct since no driver needs per-session hook file generation.
- [ADR-005: Zero Workdir Pollution via CLI Flags and Environment Variables](adrs/adr-005.md) — Pass system prompts and runtime config via CLI flags and env vars instead of writing files to user's workdir.

## Testing Strategy

### Unit Tests

- `internal/plugins/install_test.go`: Test `DetectDrivers()` with mocked `exec.LookPath`. Test `Install()` and `Uninstall()` with temp directories simulating global config paths. Test Codex hooks.json merge (empty file, existing hooks, existing AGH hooks, malformed JSON).
- All driver `*_test.go`: Verify `buildCommand()` produces correct flags. Verify `buildEnv()` sets `AGH_*` variables. Verify `Start()` does not write any files to workdir. Verify `ParseHookEvent()` still works (unchanged).

### Integration Tests

- `internal/cli/hooks_test.go`: Update to use `AGH_*` env vars. Verify `agh hook-event` reads new variable names.
- Full lifecycle: `agh install` → agent start → hook event → `agh hook-event` → `HookRouter` receives event. Verify no files in workdir after agent stop.
