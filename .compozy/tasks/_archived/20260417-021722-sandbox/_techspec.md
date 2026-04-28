# TechSpec: Execution Sandbox Abstraction + Daytona Provider

## Executive Summary

This change introduces an execution sandbox abstraction layer that decouples AGH's ACP runtime from the local OS, then ships Daytona as the first remote provider.

Today AGH hardcodes local subprocess spawning (`internal/acp/client.go:143-191`), local file IO (`internal/acp/handlers.go:180-213`), and local terminal creation throughout the ACP layer. This change extracts three interfaces — `Launcher`, `ToolHost`, and `Provider` — implements `local` as the first provider preserving current behavior, then implements `daytona` as the first remote provider using SSH for ACP transport and the Daytona Go SDK for filesystem and lifecycle operations.

The primary trade-off is increased indirection in the ACP hot path (every file read/write goes through an interface call) in exchange for a clean provider model that supports local, Daytona, and future backends without conditional branches in `acp` or `session`.

This spec assumes the newer extensibility runtime is the canonical extension control plane. Sandbox providers remain daemon-native operational subsystems, but their extension-facing seams ride on the shared extension surfaces, capability grants, and resource-backed hook-binding runtime defined in `.compozy/tasks/extensibility-parity/_techspec.md`, rather than introducing a second manager-local authority path.

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────┐
│  daemon (composition root)                          │
│  - wires provider registry, injects into session    │
└──────────┬──────────────────────────────┬───────────┘
           │                              │
    ┌──────▼──────┐              ┌────────▼─────────────┐
    │   session    │              │   extension runtime  │
    │   manager    │              │ (surfaces + hooks +  │
    │              │              │   operational APIs)  │
    └──────┬──────┘              └────────┬─────────────┘
           │                              │
    ┌──────▼──────────────────────────────▼───────┐
    │              environment                     │
    │  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
    │  │  local/   │  │ daytona/ │  │ (future)  │  │
    │  │ provider  │  │ provider │  │  e2b/     │  │
    │  └────┬─────┘  └────┬─────┘  └───────────┘  │
    │       │              │                        │
    │       │     ┌────────▼────────┐               │
    │       │     │   transport     │               │
    │       │     │  ┌───────────┐  │               │
    │       │     │  │sshTransport│ │               │
    │       │     │  └───────────┘  │               │
    │       │     └─────────────────┘               │
    └───────┼───────────────┼───────────────────────┘
            │               │
     ┌──────▼──────┐  ┌─────▼──────┐
     │   acp       │  │  Daytona   │
     │   driver    │  │  Go SDK    │
     │ (protocol)  │  │  + SSH     │
     └─────────────┘  └────────────┘
```

**Data flow — session create with Daytona (authoritative lifecycle):**

1. `session.Manager.Create` resolves workspace → `ResolvedWorkspace.Sandbox` identifies the Daytona profile.
2. Session manager dispatches `sandbox.prepare` through the canonical extension hook runtime. The sync patch may deny sandbox creation or inject `env_overrides`.
3. Session manager allocates an `SandboxID`, persists `SessionSandboxMeta` in `creating` state, then calls `Provider.Prepare()` → Daytona provider creates or reattaches to a sandbox via SDK using AGH labels/tags (`agh_session_id`, `agh_sandbox_id`). Returns `Prepared` with runtime paths, provider state, and `Launcher`/`ToolHost`.
4. Session manager calls `Provider.SyncToRuntime(state, SyncReasonStart)` → streams workspace `RootDir` + `AdditionalDirs` into the sandbox as tar archives over SSH. Daytona SDK file APIs remain for point operations through `ToolHost`; directory sync is tar-first.
5. Session manager dispatches `sandbox.ready` through the canonical extension hook runtime.
6. Session manager builds `acp.StartOpts` with `Prepared.RuntimeRootDir` and `Prepared.RuntimeAdditionalDirs`.
7. `acp.Driver.Start` uses the injected `Launcher` → Daytona launcher opens SSH session to sandbox, starts ACP agent command, returns `Handle` with clean stdin/stdout pipes.
8. ACP protocol negotiation proceeds over SSH pipes — identical to local.
9. ACP inbound callbacks (`fs/read`, `fs/write`, `session/request_permission`, terminal) route through `ToolHost` → Daytona tool host uses SDK for file operations and SSH for terminals.
10. On session stop, session manager dispatches `sandbox.sync.before` through the canonical extension hook runtime, then calls `Provider.SyncFromRuntime(state, SyncReasonStop)` → streams the runtime workspace back as tar archives, applies the configured exclude policy, and writes results to the local workspace using last-write-wins semantics.
11. After sync completes, session manager dispatches `sandbox.sync.after`, then `sandbox.stop` before teardown.
12. Session manager calls `Provider.Destroy()` if `DestroyOnStop` is set, otherwise leaves sandbox for reuse.

**Data flow — session create with local (unchanged behavior):**

The flow shape is the same except the local provider skips sync, the local launcher uses `subprocess.Launch`, and the local tool host uses `os.ReadFile`/`os.WriteFile`.

## Implementation Design

### Core Interfaces

```go
// internal/sandbox/types.go

// Provider manages the lifecycle of an execution sandbox.
type Provider interface {
    Backend() Backend
    Prepare(ctx context.Context, req PrepareRequest) (Prepared, error)
    SyncToRuntime(ctx context.Context, state SessionState, reason SyncReason) error
    SyncFromRuntime(ctx context.Context, state SessionState, reason SyncReason) error
    Destroy(ctx context.Context, state SessionState) error
}

// Launcher starts an ACP-capable agent process inside a sandbox.
type Launcher interface {
    Launch(ctx context.Context, spec LaunchSpec) (Handle, error)
}

// Handle represents a running agent process.
type Handle interface {
    PID() int
    Cwd() string
    Stdin() io.WriteCloser
    Stdout() io.ReadCloser
    Stderr() string
    Done() <-chan struct{}
    Wait() error
    Stop(ctx context.Context) error
}
```

```go
// internal/acp/tool_host.go

// ToolHost abstracts ACP file, permission, and terminal operations.
// The local implementation delegates to os.ReadFile/os.WriteFile and
// the existing permissionPolicy. Remote implementations delegate to
// provider SDKs while keeping permission decisions daemon-local.
type ToolHost interface {
    // File operations
    ReadTextFile(ctx context.Context, path string) (string, error)
    WriteTextFile(ctx context.Context, path, content string) error
    ResolvePath(path string) (string, error)

    // Permission operations — approval decisions stay daemon-local,
    // but path resolution and policy enforcement route through the
    // provider so remote paths resolve correctly.
    Authorize(op PermissionOperation) error
    PermissionDecision(req acpsdk.RequestPermissionRequest) (PermissionDecision, bool)

    // Terminal operations
    CreateTerminal(ctx context.Context, req CreateTerminalRequest) (CreateTerminalResponse, error)
    KillTerminal(id string) error
    TerminalOutput(id string) (string, error)
    WaitForTerminalExit(ctx context.Context, id string) (int, error)
    ReleaseTerminal(id string) error
}
```

### Data Models

**Config — sandbox profiles:**

```go
// internal/config/config.go

type SandboxProfile struct {
    Backend     string            `toml:"backend"`
    SyncMode    string            `toml:"sync_mode,omitempty"`
    Persistence string            `toml:"persistence,omitempty"`
    RuntimeRoot string            `toml:"runtime_root,omitempty"`
    Env         map[string]string `toml:"env,omitempty"`
    Network     NetworkProfile    `toml:"network,omitempty"`
    Daytona     DaytonaProfile    `toml:"daytona,omitempty"`
}

type DaytonaProfile struct {
    APIURL      string `toml:"api_url,omitempty"`
    Target      string `toml:"target,omitempty"`
    Image       string `toml:"image,omitempty"`
    Snapshot    string `toml:"snapshot,omitempty"`
    Class       string `toml:"class,omitempty"`
    AutoStop    string `toml:"auto_stop,omitempty"`
    AutoArchive string `toml:"auto_archive,omitempty"`
}
```

`DaytonaProfile.Snapshot` is first-class startup policy, not a later optimization. If both `snapshot` and `image` are set, `snapshot` wins and `image` is treated as documentation/fallback input for operators. Snapshot creation is not automated in alpha; profiles reference a pre-baked Daytona snapshot that already contains common agent CLIs, toolchains, and dependencies. The provider logs a clear validation error when the configured snapshot is missing.

**Workspace — sandbox reference:**

```go
// internal/workspace/workspace.go

type Workspace struct {
    ID             string
    RootDir        string
    AdditionalDirs []string
    Name           string
    DefaultAgent   string
    SandboxRef string    // NEW — references a Config.Sandboxes key
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type ResolvedWorkspace struct {
    Workspace
    Config      aghconfig.Config
    Agents      []aghconfig.AgentDef
    Skills      []SkillPath
    Environment sandbox.Resolved  // NEW — resolved environment config
    ResolvedAt  time.Time
}
```

**Session — sandbox state:**

```go
// internal/store/types.go

type SessionSandboxMeta struct {
    SandboxID         string          `json:"sandbox_id,omitempty"`
    Backend               string          `json:"backend"`
    Profile               string          `json:"profile,omitempty"`
    State                 string          `json:"state,omitempty"`
    InstanceID            string          `json:"instance_id,omitempty"`
    RuntimeRootDir        string          `json:"runtime_root_dir,omitempty"`
    RuntimeAdditionalDirs []string        `json:"runtime_additional_dirs,omitempty"`
    ProviderState         json.RawMessage `json:"provider_state,omitempty"`
    SSHAccessExpiresAt    *time.Time      `json:"ssh_access_expires_at,omitempty"`
    LastSyncAt            *time.Time      `json:"last_sync_at,omitempty"`
    LastSyncError         string          `json:"last_sync_error,omitempty"`
}
```

`SandboxID` is daemon-allocated before any provider call and is written to the session metadata in `creating` state. Remote providers must apply `agh_session_id` and `agh_sandbox_id` labels/tags during create. If the provider creates a remote sandbox but the API call times out before returning, restart reconciliation lists remote sandboxes by those labels and either reattaches or destroys according to the persisted session state. `ProviderState` replaces the earlier generic `ResumeToken` placeholder and stores provider-specific reattach data such as Daytona sandbox metadata, archived state, and token issue metadata.

**Database schema:**

```sql
ALTER TABLE workspaces ADD COLUMN sandbox_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN sandbox_backend TEXT NOT NULL DEFAULT 'local';
ALTER TABLE sessions ADD COLUMN sandbox_id TEXT DEFAULT '';
ALTER TABLE sessions ADD COLUMN sandbox_instance_id TEXT DEFAULT '';
ALTER TABLE sessions ADD COLUMN sandbox_state TEXT DEFAULT '';
ALTER TABLE sessions ADD COLUMN sandbox_provider_state_json TEXT DEFAULT '';
```

### API Endpoints

**Workspace contract changes** (`internal/api/contract/contract.go`):

Environment selection is exposed through workspace CRUD — users set `sandbox_ref` when registering or updating a workspace.

```go
type CreateWorkspaceRequest struct {
    RootDir        string   `json:"root_dir"`
    Name           string   `json:"name,omitempty"`
    AddDirs        []string `json:"add_dirs,omitempty"`
    DefaultAgent   string   `json:"default_agent,omitempty"`
    SandboxRef string   `json:"sandbox_ref,omitempty"` // NEW — Config.Sandboxes key
}

type UpdateWorkspaceRequest struct {
    Name           *string   `json:"name"`
    AddDirs        *[]string `json:"add_dirs"`
    DefaultAgent   *string   `json:"default_agent"`
    SandboxRef *string   `json:"sandbox_ref"` // NEW
}

type WorkspacePayload struct {
    // ... existing fields ...
    SandboxRef string `json:"sandbox_ref,omitempty"` // NEW
}
```

Underlying domain types (`RegisterOptions`, `UpdateOptions` in `internal/workspace/resolver.go`) gain matching `SandboxRef` fields. Conversion function `WorkspacePayloadFromWorkspace()` in `internal/api/core/conversions.go` maps the new field.

CLI commands `workspace add` and `workspace edit` gain `--sandbox` flag.

**Session contract changes** (`internal/api/contract/contract.go`):

Session list/get/status responses include environment summary for operator visibility.

```go
type SessionSandboxPayload struct {
    SandboxID string `json:"sandbox_id,omitempty"`
    Backend       string `json:"backend"`
    Profile       string `json:"profile,omitempty"`
    State         string `json:"state,omitempty"`
    InstanceID    string `json:"instance_id,omitempty"`
    SyncState     string `json:"sync_state,omitempty"`
    LastSyncError string `json:"last_sync_error,omitempty"`
}

type SessionPayload struct {
    // ... existing fields ...
    Environment *SessionSandboxPayload `json:"environment,omitempty"` // NEW
}
```

`SessionInfo` in `internal/session/session.go` gains a matching `Environment` field. Conversion function `SessionPayloadFromInfo()` maps it.

**No new session creation fields** — environment comes from workspace resolution, not from the request.

**New Host API methods for extensions:**

These methods are operational Host API surfaces. They are registered through the canonical extension surfaces and grant model from the extensibility-parity runtime and are not represented as desired-state `resources/*` kinds.

| Method             | Request                                                | Response                                                                                                 | Description                                                                                 |
| ------------------ | ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `sandbox/list` | `{}`                                                   | `{environments: [{session_id, backend, profile, instance_id, state}]}`                                   | List active sandbox instances visible to the caller's granted scope.                    |
| `sandbox/info` | `{session_id: string}`                                 | `{sandbox_id, backend, profile, instance_id, runtime_root, sync_state, created_at, last_sync_error}` | Get details of a specific visible sandbox.                                              |
| `sandbox/exec` | `{session_id: string, command: string, timeout?: int}` | `{exit_code: int, stdout: string, stderr: string}`                                                       | Execute a command inside a visible sandbox. Requires explicit `sandbox.exec` grant. |

- Visibility is filtered by caller authority and maximum scope. Workspace-scoped extensions do not see unrelated sessions.
- `sandbox.exec` is gated by daemon-computed operational capability grants, not by request-declared metadata.

**Extension hooks (lifecycle):**

`sandbox.*` joins the canonical hook taxonomy, but dispatch authority comes from the resource-backed `hook.binding` runtime defined in the extensibility-parity design. `internal/hooks` still owns payload and event type declarations; `internal/session` and `internal/daemon` must not add a parallel manager-local sandbox-hook authority path. Sync hooks support `ControlPatch.Deny` to abort sync.

| Hook                      | Mode  | Payload Fields                                                                                | Patch Fields                                             |
| ------------------------- | ----- | --------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| `sandbox.prepare`     | sync  | `profile: SandboxProfile`, `workspace_id`, `backend`                                      | `Deny`, `DenyReason`, `env_overrides: map[string]string` |
| `sandbox.ready`       | async | `instance_id`, `runtime_root`, `backend`                                                      | — (observation only)                                     |
| `sandbox.sync.before` | sync  | `direction: "to_runtime"\|"from_runtime"`, `reason: SyncReason`, `file_count`                 | `Deny`, `DenyReason`, `exclude_patterns: []string`       |
| `sandbox.sync.after`  | async | `direction`, `reason`, `files_synced`, `bytes_transferred`, `duration_ms`, `errors: []string` | — (observation only)                                     |
| `sandbox.stop`        | sync  | `instance_id`, `stop_reason`, `will_destroy: bool`                                            | `Deny`, `DenyReason`                                     |

Implementation note: registering `sandbox.*` payloads in `events.go` and `payloads.go` remains necessary, but which extensions receive those events is projected by the canonical hook-binding runtime rather than hard-coded dispatcher families.

## Integration Points

### Daytona Go SDK

- **Package**: `github.com/daytonaio/daytona/libs/sdk-go/pkg/daytona`
- **Authentication**: `DAYTONA_API_KEY` environment variable or programmatic `DaytonaConfig{APIKey, APIUrl}`
- **Lifecycle**: `Client.Create()`, `Sandbox.Start()`, `Stop()`, `Archive()`, `Delete()`. Create calls must attach AGH labels/tags (`agh_session_id`, `agh_sandbox_id`) before returning control to session lifecycle.
- **Startup inputs**: Prefer `DaytonaProfile.Snapshot` when present. Use `DaytonaProfile.Image` only when no snapshot is configured.
- **Filesystem**: `FileSystemService.UploadFile()`, `DownloadFile()`, `ListFiles()` for point operations. Bulk workspace sync uses tar-over-SSH so directory transfer does not degenerate into one SDK call per file.
- **SSH access**: `POST /api/sandbox/{id}/ssh-access` → token for `golang.org/x/crypto/ssh`
- **Error handling**: Wrap SDK errors with AGH context. Timeout all SDK calls. Retry transient failures with backoff.

### SSH Transport

- **Package**: `golang.org/x/crypto/ssh`
- **Authentication**: Token-based via Daytona SSH API
- **Token lifecycle**: 60-minute default expiry. Persist `SSHAccessExpiresAt` and proactively refresh at 50% of the expiry window. On an auth failure, refresh once and retry the failed connection before surfacing an error.
- **Session model**: One SSH client per sandbox, one session per ACP agent. Bulk sync opens separate short-lived SSH sessions. ACP transport never allocates a PTY.
- **Keepalive**: SSH client-level keepalive (30s interval)

### Filesystem Sync

Session-bidirectional sync is tar-first:

- **Sync to runtime**: AGH builds one tar stream for `Workspace.RootDir` and one tar stream per `AdditionalDirs` entry, then extracts each archive into its corresponding runtime root.
- **Sync from runtime**: AGH creates tar streams from the runtime roots and applies them to the canonical local roots with last-write-wins file replacement.
- **Excludes**: Respect profile excludes plus safe defaults for generated dependency/build output such as `node_modules`, `dist`, `build`, `target`, and cache directories. Do not exclude `.git` by default; agents often need repository metadata. Operators may exclude `.git` explicitly for read-only or non-git workflows.
- **Safety**: Tar extraction rejects absolute paths, `..` traversal, and archive entries that escape the destination after symlink evaluation. Unsupported file modes are logged and skipped rather than silently misapplied.
- **ToolHost boundary**: ACP `fs/read`/`fs/write` and permission path resolution still go through `ToolHost`; they are not routed through the tar sync path.

Delta, rsync-like transfer, and watch-based sync remain future work. Alpha sync always transfers the configured roots at session boundaries.

## Security

### Environment Variable Propagation

Today ACP agent subprocesses receive all daemon env vars (via `os.Environ()` in `daemonMatchedEnv`, `acp/client.go:591-619`) plus session-specific vars (`AGH_SESSION_ID`, `AGH_SESSION_CHANNEL`, `AGH_PEER_ID`). Extensions already use a hard-coded allowlist of safe keys (`safeSubprocessEnvKeys` in `extension/manager.go:59-74`).

For remote sandboxes, env var propagation must be explicit:

- **Allowlist-based**: Only propagate `AGH_*` session vars, provider-required vars (`DAYTONA_API_KEY` is daemon-only, not propagated), and user-declared vars from the sandbox profile.
- **Profile-level env overrides**: `SandboxProfile.Env map[string]string` allows declaring env vars to inject into the sandbox.
- **Hook-based injection**: The `sandbox.prepare` hook patch includes `env_overrides` for extensions to inject vars (e.g., secrets from a vault).
- **Never propagate**: `DAYTONA_API_KEY`, `DAYTONA_JWT_TOKEN`, daemon-internal vars. The provider uses these internally but does not pass them into the sandbox.

### Network Policy Enforcement

`SandboxProfile.Network` declares intent (`AllowPublicIngress`, `AllowOutbound`, `AllowList`, `DenyList`). Provider implementations enforce what they can:

- **Daytona**: `AllowPublicIngress` maps to preview link visibility. `AllowOutbound` and deny/allow lists are best-effort — Daytona does not expose per-sandbox firewall rules. Log a warning if a policy cannot be enforced.
- **Local**: Network policy is ignored (local processes have host network access).
- **Validation**: `Provider.Prepare()` returns an error if a required policy setting is unsupported and the profile marks it as `required`. Otherwise, log and continue.

## Compatibility With Extensibility Runtime

When this sandbox design and the extensibility-parity runtime both exist, the boundary is:

- `internal/sandbox` owns operational sandbox lifecycle, transport, sync, and runtime path handling.
- `internal/resources` remains the desired-state control plane for extensibility families. This spec does not add `sandbox.*` resource kinds.
- `sandbox.*` hooks are sandbox-domain events, but extension routing and authority come from the canonical hook-binding runtime.
- `sandbox/list`, `sandbox/info`, and `sandbox/exec` remain operational Host APIs registered through `internal/extension/surfaces`, not generic resource CRUD.
- Daemon boot order becomes explicit: `cleanupOrphans` first, then canonical resource-runtime boot/reconcile, then environment reattach/cleanup for remote backends.

## Impact Analysis

| Component                               | Impact Type | Description and Risk                                                                                                                                                                | Required Action                           |
| --------------------------------------- | ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| `internal/sandbox/`                 | New         | Core sandbox abstractions and provider registry                                                                                                                                 | Create package                            |
| `internal/sandbox/local/`           | New         | Local provider wrapping current subprocess/OS behavior                                                                                                                              | Create package                            |
| `internal/sandbox/daytona/`         | New         | Daytona provider with SSH transport                                                                                                                                                 | Create package                            |
| `internal/acp/client.go`                | Modified    | Replace `spawnProcess` with injected `Launcher`. Medium risk — core launch path.                                                                                                    | Refactor subprocess spawning              |
| `internal/acp/handlers.go`              | Modified    | Replace `os.ReadFile`/`os.WriteFile`/terminal handlers with `ToolHost`. Medium risk — ACP hot path.                                                                                 | Extract tool host interface               |
| `internal/acp/permission.go`            | Modified    | Permission resolution delegates to `ToolHost.ResolvePath`. Low risk.                                                                                                                | Inject path resolver                      |
| `internal/session/manager.go`           | Modified    | Accept sandbox manager dependency. Low risk.                                                                                                                                    | Add functional option                     |
| `internal/session/manager_start.go`     | Modified    | Call `Provider.Prepare` before driver start, use runtime paths. Medium risk.                                                                                                        | Add sandbox prepare step              |
| `internal/session/manager_lifecycle.go` | Modified    | Call sync-from and destroy on stop/crash. Low risk.                                                                                                                                 | Add lifecycle hooks                       |
| `internal/workspace/workspace.go`       | Modified    | Add `SandboxRef` to `Workspace`, `Environment` to `ResolvedWorkspace`. Low risk.                                                                                                | Add fields                                |
| `internal/workspace/resolver.go`        | Modified    | Resolve environment during workspace resolution. Low risk.                                                                                                                          | Add resolution step                       |
| `internal/config/config.go`             | Modified    | Add `Environments` map and `Defaults.Sandbox`. Low risk.                                                                                                                        | Add config fields                         |
| `internal/config/merge.go`              | Modified    | Merge `environments.*` overlays. Low risk.                                                                                                                                          | Add merge logic                           |
| `internal/store/types.go`               | Modified    | Add `SessionSandboxMeta`. Low risk.                                                                                                                                             | Add type                                  |
| `internal/store/globaldb/`              | Modified    | Persist workspace sandbox_ref and session sandbox state. Low risk.                                                                                                          | Add columns                               |
| `internal/daemon/daemon.go`             | Modified    | Wire sandbox registry and providers. Low risk — composition root.                                                                                                               | Add wiring                                |
| `internal/daemon/boot.go`               | Modified    | Add sandbox reconciliation on daemon restart and order it after canonical resource-runtime boot when that runtime is enabled. Low-medium risk because boot ownership is shared. | Add ordered cleanup step in `bootRuntime` |
| `internal/api/contract/contract.go`     | Modified    | Add `SandboxRef` to workspace contracts, `Environment` to session contracts. Low risk.                                                                                          | Add fields                                |
| `internal/api/core/conversions.go`      | Modified    | Map new sandbox fields in workspace/session conversions. Low risk.                                                                                                              | Update conversion functions               |
| `internal/api/core/workspaces.go`       | Modified    | Pass `SandboxRef` through workspace CRUD handlers. Low risk.                                                                                                                    | Add field mapping                         |
| `internal/cli/workspace.go`             | Modified    | Add `--sandbox` flag to `workspace add` and `workspace edit`. Low risk.                                                                                                         | Add CLI flag                              |
| `internal/extension/surfaces`           | Modified    | Register `environment/*` operational Host APIs and their capability metadata. Low-medium risk if wired ad hoc outside the shared surfaces table.                                    | Extend shared surfaces table              |
| `internal/hooks/events.go`              | Modified    | Register 5 new `sandbox.*` hook events. Low risk.                                                                                                                               | Add events                                |
| `internal/hooks/payloads.go`            | Modified    | Add sandbox hook payload and patch types. Low risk.                                                                                                                             | Add types                                 |
| `internal/hooks/dispatch.go`            | Modified    | Add sandbox hook dispatch entrypoints without reintroducing a parallel authority path. Low-medium risk because the canonical hook-binding runtime owns routing.                 | Add taxonomy-level dispatchers only       |

## Testing Approach

### Unit Tests

- **Provider interface compliance**: Each provider (local, daytona) passes a shared test suite verifying Prepare/Sync/Destroy contracts.
- **Launcher interface compliance**: Both local and SSH launchers pass tests verifying Handle behavior (Stdin/Stdout streaming, Stop, Wait).
- **ToolHost interface compliance**: Both local and Daytona tool hosts pass tests for file read/write/resolve and terminal operations.
- **Config validation**: Sandbox profile parsing, validation, merge, and defaults cascade.
- **Session sandbox metadata**: Persistence and recovery of sandbox state in session store.
- **Tar sync service**: Archive creation/extraction, exclude handling, path traversal rejection, symlink escape rejection, unsupported file mode handling, and last-write-wins application.
- **Daytona profile validation**: Snapshot/image precedence, missing snapshot errors, and explicit env allowlist validation.
- **SSH token manager**: Refresh at 50% of expiry window, persisted expiry handling, and single auth-failure refresh retry.
- **Mock boundary**: Mock `Provider`, `Launcher`, `ToolHost` for session/ACP tests. Real implementations tested in integration.

### Integration Tests

- **Local provider end-to-end**: Create session with local sandbox, prompt, verify file IO and terminal work unchanged. Uses mock ACP server subprocess.
- **ACP refactor regression**: Verify that the Launcher/ToolHost extraction does not change observable ACP behavior for local sessions. Run existing ACP test suite against new abstractions.
- **Config resolution**: Load TOML config with sandbox profiles, verify workspace resolution picks correct environment.
- **Concurrent same-workspace divergence**: Two sessions referencing the same workspace with local provider. Both modify the same file. Verify last-write-wins semantics on session stop and that no data corruption occurs.
- **Session resume with sandbox state**: Create session with sandbox metadata, stop it, resume it. Verify `SessionSandboxMeta` is correctly persisted and restored. Verify `Prepared` result reattaches to existing sandbox (Daytona) or creates fresh (local).
- **Partially failed provider create**: Simulate a Daytona create that succeeds remotely but times out locally. Restart daemon. Verify reconciliation finds the sandbox by `agh_sandbox_id`, reattaches when the session is recoverable, or destroys when unrecoverable.
- **Daemon restart sandbox cleanup**: Simulate daemon crash with active sandbox sessions. Restart daemon. Verify orphaned sandboxes are detected via provider labels and cleaned up.
- **Daytona provider** (tagged `integration`, requires `DAYTONA_API_KEY`): Create sandbox from snapshot when configured, SSH connect, verify clean stdio, sync files via tar, refresh SSH token, destroy. Gated by environment variable availability.
- **Observability contract**: Verify logs/metrics/spans include `session_id`, `workspace_id`, `sandbox_id`, `backend`, `profile`, `instance_id`, durations, transfer sizes, and provider error kinds.

## Development Sequencing

### Build Order

1. **Core sandbox types and interfaces** (`internal/sandbox/types.go`) — no dependencies. Define `Backend`, `Provider`, `Launcher`, `Handle`, `ToolHost`, `Resolved`, `SessionState`, `PrepareRequest`, `Prepared`, `SyncReason`.

2. **Config sandbox profiles** (`internal/config/`) — depends on step 1 types. Add `SandboxProfile`, `DaytonaProfile`, `Defaults.Sandbox`, `Config.Sandboxes`. Add validation and merge logic, including `DaytonaProfile.Snapshot` precedence over `Image` and explicit environment variable allowlist handling.

3. **Workspace sandbox resolution** (`internal/workspace/`) — depends on step 1 and 2. Add `SandboxRef` to `Workspace`, `Environment` to `ResolvedWorkspace`. Add resolution in `buildResolvedWorkspace`. Add `sandbox_ref` column to workspace table.

4. **Extract ACP Launcher interface** (`internal/acp/`) — depends on step 1. Extract `spawnProcess` into `localLauncher` implementing `Launcher`. Inject `Launcher` into `acp.Driver`. Verify zero behavior change.

5. **Extract ACP ToolHost interface** (`internal/acp/`) — depends on step 1. Extract file IO handlers and terminal handlers into `localToolHost` implementing `ToolHost`. Inject `ToolHost` into ACP handler dispatch. Verify zero behavior change.

6. **Local provider** (`internal/sandbox/local/`) — depends on steps 4 and 5. Wrap `localLauncher` and `localToolHost` into a `Provider` implementation where `Prepare` is a no-op, sync is a no-op, and `Destroy` is a no-op. Provider registry with `local` as default.

7. **Session sandbox integration** (`internal/session/`) — depends on steps 3 and 6. Inject sandbox manager/registry. Allocate `SandboxID`, persist `SessionSandboxMeta` in `creating` state before provider create, call `Prepare` in `startSession`, persist returned provider state, and call sync/destroy in lifecycle hooks. Add `sandbox_id`, `sandbox_backend`, `sandbox_instance_id`, `sandbox_state`, and `sandbox_provider_state_json` to sessions table.

8. **Daemon wiring** (`internal/daemon/`) — depends on steps 6 and 7. Wire provider registry, local provider, and session manager options in the composition root.

9. **Validate Daytona SSH non-PTY** — depends on nothing (can run in parallel with steps 1-8). Spike: create a Daytona sandbox, generate SSH token, run `echo '{"test":true}' | ssh <token>@ssh.app.daytona.io cat` and verify output matches input byte-for-byte. **This is a blocking gate for steps 10-11.** If SSH gateway forces PTY, switch to WebSocket sidecar transport (the internal transport interface from step 10 enables this swap).

10. **Daytona SSH transport** (`internal/sandbox/daytona/`) — depends on steps 1 and 9. Implement `sshTransport` using `golang.org/x/crypto/ssh`. Token management via Daytona REST API, with persisted expiry and proactive refresh at 50% of the token lifetime. Implement provider-internal `transport` interface.

11. **Daytona provider** (`internal/sandbox/daytona/`) — depends on steps 1, 10. Implement `Provider` (sandbox create/start/stop/archive/delete), `Launcher` (via SSH transport), `ToolHost` (via Daytona SDK filesystem + SSH terminals), snapshot-aware sandbox creation, AGH label/tag attachment, and session-bidirectional tar sync over SSH.

12. **Daemon restart sandbox cleanup** (`internal/daemon/boot.go`) — depends on step 8. Add sandbox reconciliation in `bootRuntime` after `cleanupOrphans` and after the canonical resource-runtime boot/reconcile phase when that runtime is enabled. Load persisted `SessionSandboxMeta` for all sessions. For sessions in non-terminal states with remote backends: attempt reattach via `Provider.Prepare()` using `SandboxID`, `InstanceID`, and `ProviderState`. If local metadata is missing `InstanceID` but remote labels contain the daemon-allocated `SandboxID`, reconcile the partial create by attaching the sandbox to the session or destroying it if the session is unrecoverable. Follows existing pattern of `observer.Reconcile()`.

13. **Sandbox extension hooks + operational Host APIs** (`internal/hooks/`, `internal/extension/surfaces`) — depends on step 8 plus the extensibility-parity foundation for extension surfaces, grant computation, and resource-backed `hook.binding` dispatch. Register 5 lifecycle hooks (`sandbox.prepare`, `sandbox.ready`, `sandbox.sync.before`, `sandbox.sync.after`, `sandbox.stop`) with payload/patch types in `internal/hooks`, but route extension delivery through the canonical hook runtime rather than a manager-local path. Add 3 operational Host API methods (`sandbox/list`, `sandbox/info`, `sandbox/exec`) through the shared surfaces table. Requires changes across `events.go`, `payloads.go`, `dispatch.go`, `matcher.go`, and the extension surfaces metadata.

14. **End-to-end Daytona integration test** — depends on steps 11 and 13. Tagged `integration`. Create workspace with Daytona profile, create session, verify SSH transport, verify file sync, verify session stop cleanup.

15. **Terminal/process follow-up note** — do not block alpha on a full process manager. Capture the future design in a follow-up spec: PTY I/O should use a streaming transport, command execution should stay request/response, resize should be a separate control call, and process lifetime should eventually be decoupled from client connection lifetime.

### Technical Dependencies

- `archive/tar` — Go standard library tar creation/extraction for bulk sync. No external sync dependency in alpha.
- `golang.org/x/crypto/ssh` — SSH client. Stable, part of Go extended standard library.
- `github.com/daytonaio/daytona/libs/sdk-go` — Daytona Go SDK. Pin to specific commit. Isolate behind internal adapter.
- `DAYTONA_API_KEY` — Required for Daytona provider. Stored as daemon environment variable or config.
- Sandbox hook and Host API work must reuse the extensibility-parity foundation for `internal/extension/surfaces`, daemon-computed grants, and resource-backed `hook.binding` dispatch. Do not add a sandbox-specific extension registry or manager-local routing path.

## Monitoring and Observability

- **Metrics**: Environment prepare duration, sync duration (to/from), sync file count/size, SSH connection lifetime, provider errors by type.
- **Log events**: `sandbox.prepare.start/complete/error`, `sandbox.sync.start/complete/error`, `sandbox.transport.connect/disconnect/error`, `sandbox.destroy.start/complete/error`. Structured fields: `backend`, `profile`, `instance_id`, `session_id`, `duration_ms`.
- **Tracing**: Emit optional OpenTelemetry-compatible spans through the existing observability layer. Required span names: `sandbox.prepare`, `sandbox.sync.to_runtime`, `sandbox.sync.from_runtime`, `sandbox.transport.ssh.connect`, `sandbox.transport.ssh.refresh_token`, and `sandbox.destroy`. Required attributes: `session_id`, `workspace_id`, `sandbox_id`, `backend`, `profile`, `instance_id`, `duration_ms`, `files`, `bytes`, and `error_kind` when applicable.
- **No anonymous telemetry**: This feature does not add outbound product telemetry. All spans/logs/metrics stay local unless the operator has explicitly configured an exporter in the daemon.
- **Health**: Provider health exposed in daemon status. SSH connection state tracked per active session.
- **Status/list surfaces**: Session list/get/status responses include `environment` field (`sandbox_id`, `backend`, `profile`, `state`, `instance_id`) for operator visibility. CLI `session list` shows backend/state columns. CLI `session info` shows full sandbox details including sync state and last sync error.

## Technical Considerations

### Key Decisions

1. **Daemon-native providers, not extensions** (ADR-001). Sandbox providers run in-process for zero-latency tool host operations. Extension hooks provide extensibility.

2. **SSH as primary transport** (ADR-002). SSH non-PTY gives clean stdio pipes matching AGH's existing interface. Internal transport abstraction enables migration to WebSocket sidecar.

3. **Session-scoped sandbox with copy-on-start/collect-on-stop** (ADR-003). One sandbox per session. No shared sandboxes. No turn-level sync. Last-write-wins on divergence.

4. **Workspace-scoped sandbox selection**. `Workspace.SandboxRef` → `Config.Defaults.Sandbox` → `local`. Consistent with existing default-agent override model.

5. **ToolHost abstracts complete ACP tool surface**. File IO, permission policy (`Authorize`, `PermissionDecision`), path resolution, and terminal operations all route through the provider's ToolHost implementation. Permission approval decisions (interactive user prompts) stay daemon-local — the ToolHost only handles policy evaluation and path resolution so remote paths work correctly. This avoids the split-brain risk where terminal edits bypass `fs/write` permissions: the permission policy root is the runtime path, not the local path.

6. **Local path vs runtime path separation**. `Workspace.RootDir` stays canonical. `SessionState.RuntimeRootDir` tracks the sandbox-internal path. ACP uses runtime paths for agent communication.

7. **Sandbox hooks and Host APIs layer onto the canonical extension runtime**. The sandbox feature defines `sandbox.*` payloads/events and operational methods, but grants, delivery, and capability exposure come from the shared extension surfaces and hook-binding runtime rather than a sandbox-specific registry.

8. **Environment operations remain operational, not desired state**. `sandbox/list`, `sandbox/info`, and `sandbox/exec` stay family-specific Host APIs. They are not modeled as `internal/resources` kinds.

9. **Tar sync is first-class for directory transfer**. The Daytona SDK remains the point-operation file API, but session-boundary workspace sync uses tar streams over SSH to avoid one remote API call per file.

10. **Provider create is idempotent by daemon identity**. AGH allocates `SandboxID` before provider create, labels remote sandboxes with it, and persists provider-specific reattach state in `ProviderState` instead of relying on a vague resume token.

11. **Snapshots are explicit profile inputs**. Daytona snapshots are treated as a first-class cold-start strategy. Alpha consumes pre-baked snapshots; it does not build or mutate snapshots automatically.

### Known Risks

| Risk                                                    | Likelihood | Impact                               | Mitigation                                                                                                                                                                                      |
| ------------------------------------------------------- | ---------- | ------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Daytona SSH gateway forces PTY                          | Low        | High — blocks SSH transport          | Validate early with spike (step 9, blocking gate). Fall back to WebSocket sidecar via internal transport interface.                                                                             |
| SSH token expiry during long sessions                   | Medium     | Medium — agent disconnects           | Persist token expiry, refresh at 50% of the expiry window, and retry once after auth failure with a fresh token.                                                                                |
| Large workspace sync is slow                            | Medium     | Medium — session start latency       | Use tar-over-SSH as the primary directory sync path. Add configurable exclude patterns for generated dependency/build output. Log sync duration, file count, and bytes.                         |
| Tar extraction path traversal                           | Low        | High — local file corruption         | Reject absolute paths, `..`, and symlink-resolved escapes during extraction. Add unit tests with malicious archives.                                                                            |
| Missing or stale Daytona snapshot                       | Medium     | Medium — sandbox start fails         | Validate `DaytonaProfile.Snapshot` during prepare, return an actionable error, and allow operators to fall back to `image` by clearing `snapshot`.                                              |
| Daytona SDK breaking changes                            | Low        | Medium — provider breaks             | Pin SDK to commit hash. Wrap SDK behind internal adapter.                                                                                                                                       |
| ToolHost extraction breaks ACP behavior                 | Low        | High — regression                    | Step 5 runs existing ACP test suite against new abstraction. Zero behavior change required.                                                                                                     |
| Concurrent workspace divergence (silent overwrite)      | Medium     | Medium — data loss                   | Log warning when multiple active sessions share a workspace with remote sandboxes. Emit audit log on collect-on-stop with file list. Document last-write-wins semantics in user-facing docs. |
| Provider create succeeds remotely but times out locally | Medium     | Medium — orphan or duplicate sandbox | Persist `SandboxID` before create and tag remote sandboxes. Restart reconciliation attaches or destroys by `agh_sandbox_id` even when `InstanceID` was not written locally.             |
| Orphaned remote sandboxes after daemon crash            | Medium     | Medium — billing cost                | Daemon restart reconciliation (step 12) detects and cleans up orphaned sandboxes by provider label. Later add a periodic idle sweeper using `AutoStop`/`AutoArchive` policy.                    |
| Env var leakage to remote sandbox                       | Low        | High — security                      | Allowlist-based env propagation. Never propagate `DAYTONA_API_KEY` or daemon-internal secrets. Profile-level `env` map for explicit injection.                                                  |
| Terminal API grows beyond ACP compatibility shape       | Medium     | Medium — awkward remote UX           | Keep current `ToolHost` terminal methods for alpha. Design a separate process/PTY manager in a follow-up spec before adding rich interactive terminal UX.                                       |

## Architecture Decision Records

- [ADR-001: Daemon-Native Environment Providers with Extension Hooks](adrs/adr-001.md) — Sandbox providers are in-process daemon subsystems, not extensions, with lifecycle hooks exposed to the extension system.
- [ADR-002: SSH as Primary Transport for Daytona](adrs/adr-002.md) — SSH non-PTY provides clean stdio for ACP JSON-RPC, with provider-internal transport abstraction for future WebSocket migration.
- [ADR-003: Session-Scoped Sandbox with Copy-on-Start / Collect-on-Stop Sync](adrs/adr-003.md) — One sandbox per session, session-bidirectional sync, last-write-wins divergence policy.
