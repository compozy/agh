# AGH Sandbox / Execution Environment Architecture Research

I read every source area requested in Section 1 before drafting this report:

- `internal/workspace/workspace.go`
- `internal/workspace/resolver.go`
- `internal/workspace/resolver_crud.go`
- all non-test files under `internal/acp/`
- all non-test files under `internal/session/`
- all non-test files under `internal/config/`
- `DESIGN.md`
- `AGENTS.md`

This report is opinionated. The short version is:

- AGH already has the correct high-level seam at `session.AgentDriver`, but the real execution backend is still hardcoded inside `internal/acp`.
- The right abstraction is not "sandbox" but "execution environment", with `local` as a first-class provider and Daytona/E2B as optional providers.
- The first provider to ship should be Daytona, not E2B, because Daytona maps better to AGH's workspace model and has an official Go SDK surface for sandboxes, filesystem, processes, preview links, and lifecycle.
- E2B is still valuable, but it fits better as a second provider, especially for transient automation-style workloads.

## 1. Current Execution Model Analysis

### 1.1 What AGH does today

AGH currently executes agents as local subprocesses speaking ACP over stdio.

The create path is:

1. A transport request enters through the shared contract. `contract.CreateSessionRequest` carries `agent_name`, `workspace`, `workspace_path`, and `channel` (`internal/api/contract/contract.go:13-20`).
2. The HTTP path calls `BaseHandlers.CreateSession`, which validates the request and forwards it to `h.Sessions.Create(...)` (`internal/api/core/handlers.go:202-233`). The CLI path converges on the same request shape through `client.CreateSession(...)` from `newSessionCreateCommand` (`internal/cli/session.go:35-77`).
3. The daemon composition root creates the session manager in `internal/daemon/daemon.go:377-388`, injecting the workspace resolver, prompt assembler, MCP resolver, hooks, and notifier.
4. `session.Manager.Create` delegates to `prepareCreateStart` and then `startSession` (`internal/session/manager_lifecycle.go:17-29`).
5. `prepareCreateStart` resolves the workspace, resolves the agent name, allocates the session ID, and builds a `sessionStartSpec` (`internal/session/manager_start.go:38-70`).
6. `resolveCreateWorkspace` either resolves an existing workspace or auto-registers one from a path (`internal/session/manager_workspace.go:14-39`).
7. `workspace.Resolver.Resolve` loads the persisted workspace row, refreshes the canonical root, scans the workspace, and builds a cached `ResolvedWorkspace` snapshot (`internal/workspace/resolver.go:79-155`).
8. `buildResolvedWorkspace` loads the merged config, applies the workspace default-agent override, loads agent definitions, and merges skill paths (`internal/workspace/resolver.go:218-242`).
9. `scanWorkspace` snapshots global config, workspace config, MCP sidecars, and then scans workspace root, additional dirs, and global home for agents and skills in precedence order (`internal/workspace/scanner.go:37-75`, `internal/config/agent.go:105-152`).
10. `startSession` resolves the chosen agent from the resolved workspace, assembles the startup prompt, appends the bundled network skill, resolves the agent through config, resolves MCP servers, opens the session DB, writes session metadata, builds `acp.StartOpts`, and starts the driver (`internal/session/manager_start.go:101-219`).
11. `Config.ResolveAgent` merges `AgentDef` plus provider defaults into a concrete runtime command, model, permissions, and MCP server set (`internal/config/provider.go:101-162`).
12. `acp.Driver.Start` normalizes `StartOpts`, spawns the process, initializes ACP, and either creates or loads the ACP session (`internal/acp/client.go:119-140`).
13. `spawnProcess` shells out through `subprocess.Launch(...)`, using `normalized.Cwd` as the OS working directory and a permission policy rooted at that same directory (`internal/acp/client.go:143-191`).
14. `createSession` / `loadSession` send ACP `session/new` or `session/load`, passing `cwd`, MCP servers, and top-level `additional_dirs` (`internal/acp/client.go:227-277`; wire types in `internal/acp/handlers.go:38-56`).
15. During runtime, ACP callbacks from the agent are served by the daemon itself. `handleInbound` routes `fs/read`, `fs/write`, `session/request_permission`, and terminal operations (`internal/acp/handlers.go:113-152`).
16. Those handlers currently operate directly on the daemon host filesystem and host processes. `handleReadTextFile` / `handleWriteTextFile` use `os.ReadFile` / `os.WriteFile`, and terminal creation uses the local terminal manager (`internal/acp/handlers.go:180-213`, plus terminal handlers later in that file).
17. File access and permission requests are governed by a daemon-local `permissionPolicy` rooted at the workspace root (`internal/acp/permission.go:48-181`).

The prompt path is separate but consistent:

1. `Manager.PromptWithOpts` validates the request, looks up the live session, persists the user event, then calls `m.driver.Prompt(...)` (`internal/session/manager_prompt.go:31-117`).
2. `pumpPrompt` mirrors ACP events into storage, notifiers, and turn-finalization logic (`internal/session/manager_prompt.go:157-180` and the rest of that file).

### 1.2 The current execution backend is implicit, not explicit

AGH already has a good lifecycle abstraction at the session layer:

- `session.AgentDriver` is the protocol/lifecycle interface the session manager consumes (`internal/session/interfaces.go:193-199`).
- `session.Manager` already receives its dependencies by injection, and `daemon/` is the composition root (`internal/session/manager.go:56-81`, `internal/daemon/daemon.go:377-388`).

But the actual execution environment is still implicit and hardcoded:

- `acp.Driver.spawnProcess` assumes the runtime is a local subprocess (`internal/acp/client.go:143-191`).
- `permissionPolicy` assumes the authoritative filesystem is local and rooted at `Cwd` (`internal/acp/permission.go:94-181`).
- `handleInbound` assumes file IO and terminal sessions happen on the daemon host (`internal/acp/handlers.go:113-213`).

So the real backend today is not just "ACP". It is:

`local OS process + local filesystem + daemon-owned PTYs + ACP over stdio`

That matters because Daytona and E2B do not just replace `subprocess.Launch(...)`. They also change where files live, where terminals run, and how ACP callbacks should be served.

### 1.3 Where a sandbox provider should actually plug in

The correct plug-in point is below `session.startSession(...)` and inside the ACP runtime substrate.

Specifically:

- `session` should stay responsible for choosing the workspace, resolving the agent, assembling the startup prompt, and persisting session metadata.
- `acp` should stay responsible for ACP protocol negotiation, prompt streaming, and event translation.
- A new execution-environment layer should own:
  - provisioning or reusing the runtime environment
  - synchronizing the workspace into that environment
  - starting the ACP-capable agent command inside that environment
  - serving ACP file/terminal callbacks against that environment

In code terms, the most important hardcoded sites are:

- `internal/session/manager_start.go:189-209`
  - this is where a local `acp.StartOpts` is constructed and handed to the driver
- `internal/acp/client.go:143-191`
  - this is where AGH assumes "execution = local subprocess"
- `internal/acp/handlers.go:113-213`
  - this is where AGH assumes "tool host = local OS"

### 1.4 Existing interfaces that can be extended vs what is new

Useful existing seams:

- `workspace.WorkspaceResolver` already produces a runtime snapshot for a workspace (`internal/workspace/workspace.go:54-58`).
- `session.AgentDriver` already isolates session lifecycle from the concrete ACP implementation (`internal/session/interfaces.go:193-199`).
- `Config.ResolveAgent(...)` already converts `AgentDef` into an effective runtime launch plan (`internal/config/provider.go:101-162`).
- `PromptAssembler`, `SkillRegistry`, and `MCPResolver` already keep session startup composition outside the session core (`internal/session/interfaces.go:211-224`).

What is missing and must be added:

- a first-class execution-environment model in config and workspace resolution
- provider lifecycle and sync abstractions
- an ACP launcher abstraction so `acp.Driver` does not assume local subprocesses
- an ACP tool-host abstraction so filesystem and terminal operations can target a remote environment
- session metadata for provider runtime state

### 1.5 Data flow map

Current create flow:

```text
User (CLI or HTTP)
  -> contract.CreateSessionRequest
  -> BaseHandlers.CreateSession / CLI client
  -> session.Manager.Create
  -> resolveCreateWorkspace / workspace.Resolver.Resolve
  -> load Config + Agents + Skills into ResolvedWorkspace
  -> session.startSession
  -> Config.ResolveAgent
  -> acp.StartOpts{Command, Cwd, AdditionalDirs, Env, MCPServers, Permissions, SystemPrompt}
  -> acp.Driver.Start
  -> subprocess.Launch (local)
  -> ACP initialize + session/new or session/load
  -> live ACP prompt stream
```

Current runtime callback flow:

```text
Agent (ACP runtime)
  -> ACP request over stdio
  -> acp.AgentProcess.handleInbound
  -> local file IO / permission policy / local terminal manager
  -> response over stdio
```

Recommended future flow:

```text
User
  -> session.Manager.Create
  -> workspace.Resolver.Resolve
  -> environment.Manager.ResolveAndPrepare
  -> acp.Driver.Start via injected Launcher
  -> local or remote ACP runtime
  -> ACP callbacks routed through injected ToolHost
  -> session prompt/event pipeline unchanged
```

## 2. Provider Capability Matrix

### 2.1 Summary table

| Dimension                               | Daytona                                                                                                                                                                                                           | E2B                                                                                                                                                                                                                                      | Architectural reading for AGH                                                                                     |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Provisioning model                      | Official Go SDK, REST API, CLI, and SDK docs for sandboxes (`https://www.daytona.io/docs/en/go-sdk/`, `https://www.daytona.io/docs/en/sandboxes/`)                                                                | Docs emphasize JS/Python SDKs, CLI, and HTTP-style APIs; I did not find official Go SDK docs in the reviewed surface (`https://e2b.dev/docs`, `https://e2b.dev/docs/sdk-reference/js-sdk/v1.2.1/sandbox`)                                | Daytona is lower-friction for a Go daemon                                                                         |
| Lifecycle                               | Create, start, stop, archive, delete sandboxes; persistent lifecycle is explicit in docs (`https://www.daytona.io/docs/en/go-sdk/daytona/`, `https://www.daytona.io/changelog/api-rate-limits-sandbox-archiving`) | Create sandboxes, run commands, PTY sessions, templates, hibernation/resume-oriented docs (`https://e2b.dev/docs`, `https://e2b.dev/docs/template/quickstart`, `https://e2b.dev/docs/sandbox/pty`)                                       | Daytona maps better to workspace-like persistent environments; E2B maps better to transient runtime sandboxes     |
| Filesystem model                        | Official docs surface filesystem services and local-dir/file inclusion in image build context (`https://www.daytona.io/docs/en/go-sdk/daytona/`)                                                                  | Filesystem is explicit, but upload/download is centered on sandbox-local paths and docs note multiple files currently need separate uploads (`https://e2b.dev/docs/filesystem`, `https://e2b.dev/docs/quickstart/upload-download-files`) | Daytona is better for whole-workspace flows; E2B is workable but sync is more painful                             |
| Process / terminal model                | Official Go SDK docs expose process execution and PTY support (`https://www.daytona.io/docs/en/go-sdk/daytona/`, `https://www.daytona.io/docs/en/pty/`)                                                           | Commands and PTY APIs are clearly documented (`https://e2b.dev/docs`, `https://e2b.dev/docs/sandbox/pty`, `https://e2b.dev/docs/cli/exec-command`)                                                                                       | Both can run commands; Daytona looks stronger for long-lived Go-native integration                                |
| Network model                           | Preview links exist and private previews can require `x-daytona-preview-token` (`https://www.daytona.io/docs/en/go-sdk/daytona/`, `https://www.daytona.io/docs/tools/api/`)                                       | Public URLs are explicit; public access can be restricted with `allowPublicTraffic`; outbound deny rules are documented (`https://e2b.dev/docs/sandbox/internet-access`)                                                                 | E2B has clearer documented agent-style traffic controls; Daytona is still sufficient for preview/service exposure |
| Auth model                              | `DAYTONA_API_KEY`, JWT, `DAYTONA_API_URL` are documented (`https://www.daytona.io/docs/en/go-sdk/`, `https://www.daytona.io/docs/api-keys/`)                                                                      | `E2B_API_KEY` is documented; secured sandboxes and traffic tokens are documented (`https://e2b.dev/docs`, `https://e2b.dev/docs/filesystem/upload`, `https://e2b.dev/docs/sandbox/internet-access`)                                      | Both are fine; both fit env-var-based daemon config                                                               |
| Pricing                                 | Usage-based, per-second pricing, free compute credits (`https://www.daytona.io/pricing`)                                                                                                                          | Free hobby tier plus usage-based pricing; per-second billing; concurrency/session caps documented (`https://e2b.dev/pricing`)                                                                                                            | Both can fit optional provider usage, but neither should become AGH's default                                     |
| Fit for AGH local-first workspace model | Better                                                                                                                                                                                                            | Worse                                                                                                                                                                                                                                    | Daytona should be the first remote provider                                                                       |

### 2.2 Detailed comparison

#### Daytona

What stands out from the official docs:

- Official Go SDK documentation exists and is not an afterthought. The docs explicitly describe a Go SDK for interacting with Daytona sandboxes (`https://www.daytona.io/docs/en/go-sdk/`).
- The Go SDK surface includes filesystem, process execution, PTY, preview-link generation, and sandbox lifecycle operations (`https://www.daytona.io/docs/en/go-sdk/daytona/`).
- The sandbox model is persistent enough to resemble AGH's workspace abstraction: sandboxes can be started, stopped, archived, and deleted, and preview links can be generated per sandbox (`https://www.daytona.io/docs/en/go-sdk/daytona/`, `https://www.daytona.io/changelog/api-rate-limits-sandbox-archiving`).
- Daytona pricing is usage-based and per-second, with free credits (`https://www.daytona.io/pricing`).

For AGH, this means Daytona is the better match for:

- long-lived user workspaces
- session resume flows
- keeping one remote environment attached to a workspace
- a single-binary Go implementation with minimal polyglot glue

#### E2B

What stands out from the official docs:

- E2B is very explicitly designed for agent sandboxes, fast secure VMs, command execution, tools, and public URLs (`https://e2b.dev/docs`, `https://e2b.dev/docs/sandbox/internet-access`).
- Filesystem upload/download is clear, but the docs are centered on sandbox-local files, not whole local-directory mirroring (`https://e2b.dev/docs/filesystem`, `https://e2b.dev/docs/filesystem/upload`, `https://e2b.dev/docs/filesystem/download`).
- The quickstart docs explicitly say that if you want to upload multiple files you currently upload each one separately (`https://e2b.dev/docs/quickstart/upload-download-files`). That is a real signal that "mirror a full AGH workspace tree" is not the happy path.
- PTY and command execution are well documented (`https://e2b.dev/docs/sandbox/pty`, `https://e2b.dev/docs/cli/exec-command`).
- Public service exposure is explicit, and traffic restrictions are a first-class documented concern (`https://e2b.dev/docs/sandbox/internet-access`).
- Pricing is usage-based with concurrency/session limits spelled out (`https://e2b.dev/pricing`).

For AGH, this means E2B is attractive for:

- transient automation or system sessions
- strongly isolated execution
- agent-native workflows where the remote sandbox is the main runtime value

But E2B is weaker for AGH's current workspace semantics because AGH workspaces are local directory registrations with local config, local discovery, and local file authority. E2B's docs do not suggest that full local workspace mirroring is the easy path.

### 2.3 Recommendation

My recommendation is:

1. Design the abstraction around `local`, `daytona`, and `e2b` from day one.
2. Ship only `local` plus Daytona first.
3. Add E2B second, initially targeting explicit opt-in environments and likely automation/system use cases before broad interactive coding use.

The deciding factors are:

- Daytona's official Go SDK surface
- Daytona's more workspace-like lifecycle
- E2B's weaker story for synchronizing a whole local multi-directory workspace

Important inference: the E2B "Go SDK availability" gap is an inference from the official docs surface I reviewed. I found JS/Python/CLI/HTTP-oriented docs, but not official Go SDK docs.

## 3. Proposed Architecture Design

### 3.0 Recommendation in one sentence

Add a new `internal/environment/` package that resolves a workspace-selected execution environment into a prepared runtime, keep ACP as the wire protocol, replace `acp`'s hardcoded local subprocess/tool host with injected launcher and tool-host abstractions, and keep the workspace's local `RootDir` as the canonical AGH identity even when execution is remote.

### 3.1 3a. Core abstraction

I would model this as "execution environments", not "sandbox providers", because `local` should implement the same contract as Daytona and E2B.

The provider abstraction should own three responsibilities:

- prepare or reuse a runtime environment
- synchronize workspace files into and out of that environment
- start the ACP-capable agent command and expose an ACP-compatible transport/tool host

I would use the following interfaces.

```go
// internal/environment/types.go
package environment

import (
	"context"
	"time"
)

type Backend string

const (
	BackendLocal   Backend = "local"
	BackendDaytona Backend = "daytona"
	BackendE2B     Backend = "e2b"
)

type SyncMode string

const (
	SyncModeNone               SyncMode = "none"
	SyncModeSessionBidirectional SyncMode = "session-bidirectional"
	SyncModeTurnBidirectional    SyncMode = "turn-bidirectional"
)

type PersistenceMode string

const (
	PersistenceTransient PersistenceMode = "transient"
	PersistenceReuse     PersistenceMode = "reuse"
	PersistenceArchive   PersistenceMode = "archive"
)

type Resolved struct {
	Name            string
	Backend         Backend
	SyncMode        SyncMode
	Persistence     PersistenceMode
	RuntimeRootDir  string
	DestroyOnStop   bool
	Network         NetworkPolicy
	Daytona         *DaytonaConfig
	E2B             *E2BConfig
}

type NetworkPolicy struct {
	AllowPublicIngress bool
	AllowOutbound      bool
	AllowList          []string
	DenyList           []string
}

type SessionState struct {
	Backend               Backend
	Profile               string
	InstanceID            string
	RuntimeRootDir        string
	RuntimeAdditionalDirs []string
	ResumeToken           string
	PreparedAt            time.Time
}

type PrepareRequest struct {
	SessionID           string
	WorkspaceID         string
	LocalRootDir        string
	LocalAdditionalDirs []string
	Environment         Resolved
	AgentCommand        string
	AgentEnv            []string
	ResumeACPState      string
}

type Prepared struct {
	State                SessionState
	RuntimeRootDir       string
	RuntimeAdditionalDirs []string
	Launch               LaunchSpec
}

type SyncReason string

const (
	SyncReasonStart SyncReason = "start"
	SyncReasonTurn  SyncReason = "turn"
	SyncReasonStop  SyncReason = "stop"
	SyncReasonCrash SyncReason = "crash"
)

type Provider interface {
	Backend() Backend
	Prepare(ctx context.Context, req PrepareRequest) (Prepared, error)
	SyncToRuntime(ctx context.Context, prepared Prepared, reason SyncReason) error
	SyncFromRuntime(ctx context.Context, state SessionState, reason SyncReason) error
	Destroy(ctx context.Context, state SessionState) error
}
```

Then ACP gets an explicit transport/tool-host seam:

```go
// internal/acp/launcher.go
package acp

import (
	"context"
	"io"
)

type ToolHost interface {
	ReadTextFile(ctx context.Context, req ReadTextFileRequest) (ReadTextFileResponse, error)
	WriteTextFile(ctx context.Context, req WriteTextFileRequest) (WriteTextFileResponse, error)
	RequestPermission(ctx context.Context, req RequestPermissionRequest) (RequestPermissionResponse, error)
	CreateTerminal(req CreateTerminalRequest) (CreateTerminalResponse, error)
	KillTerminal(req KillTerminalRequest) (KillTerminalResponse, error)
	TerminalOutput(req GetTerminalOutputRequest) (GetTerminalOutputResponse, error)
	WaitForTerminalExit(ctx context.Context, req WaitForTerminalExitRequest) (WaitForTerminalExitResponse, error)
	ReleaseTerminal(req ReleaseTerminalRequest) (ReleaseTerminalResponse, error)
}

type Launcher interface {
	Launch(ctx context.Context, spec LaunchSpec) (Handle, error)
}

type Handle interface {
	PID() int
	Cwd() string
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Stderr() string
	Done() <-chan struct{}
	Wait() error
	Stop(ctx context.Context) error
	ToolHost() ToolHost
}
```

Why this split is the right one:

- `session` remains orchestration-focused.
- `acp` remains protocol-focused.
- `environment` becomes the place where provider-specific lifecycle and sync complexity lives.
- `local` becomes just another provider, which avoids a permanent branch explosion in `acp` and `session`.

### 3.2 3b. Integration points

#### New package: `internal/environment/`

Create a new package `internal/environment/`.

Responsibility:

- define `Backend`, profile models, resolved environment config, and per-session environment state
- own provider registry and provider selection
- implement the `local` provider
- host Daytona and E2B provider implementations behind the same contract

Why new package:

- "sandbox" is too vendor-specific
- AGH needs to represent both local and remote execution backends
- environment state spans config, workspace resolution, session lifecycle, and ACP launch

#### Changes to `workspace`

Files affected:

- `internal/workspace/workspace.go:29-58`
- `internal/workspace/resolver.go:17-30`, `218-242`, `245-263`
- `internal/workspace/resolver_crud.go:12-18`, `60-102`, `133-186`

Changes:

- add `EnvironmentRef string` to `Workspace`
- add `Environment environment.Resolved` to `ResolvedWorkspace`
- add `EnvironmentRef` to `RegisterOptions` / `UpdateOptions`
- when resolving a workspace, compute `ResolvedWorkspace.Environment`
- include `EnvironmentRef` in cache reuse checks

This keeps environment choice aligned with the existing workspace registration model.

#### Changes to `config`

Files affected:

- `internal/config/config.go:37-167`, `198-260`
- `internal/config/merge.go:15-32`, `184-205`
- `internal/config/provider.go:101-162`
- `internal/config/agent.go:105-152`

Changes:

- add `Defaults.Environment`
- add `Config.Environments map[string]EnvironmentProfile`
- add `ResolveEnvironment(...)` and validation
- extend overlay merging for `[environments.*]`

Important design choice:

- do not put sandbox/environment selection on `AgentDef`
- keep `AgentDef` responsible for the ACP agent command/model/tools
- keep environment selection workspace-scoped, with config-defined reusable profiles

#### Changes to `acp`

Files affected:

- `internal/acp/types.go:45-56`
- `internal/acp/client.go:119-191`
- `internal/acp/handlers.go:113-305`
- `internal/acp/permission.go:94-181`

Changes:

- extend `StartOpts` with environment/runtime state needed by the launcher
- replace hardcoded `subprocess.Launch(...)` inside `spawnProcess` with injected `Launcher`
- replace direct file/terminal handling with `ToolHost`
- keep ACP session negotiation unchanged

This is the most important technical refactor. Without it, Daytona/E2B support will turn into provider-specific logic scattered throughout `acp`.

#### Changes to `session`

Files affected:

- `internal/session/manager.go:56-81`, `154-159`, `196-267`
- `internal/session/interfaces.go:193-245`
- `internal/session/manager_start.go:20-36`, `101-219`
- `internal/session/manager_workspace.go:14-65`
- `internal/session/manager_lifecycle.go:37-99`
- `internal/session/manager_prompt.go:157-180`
- `internal/session/session.go:62-91`

Changes:

- inject an environment manager/runtime factory through `NewManager(...)`
- resolve environment before driver start
- persist `SessionEnvironmentState` in metadata before the runtime becomes active
- on turn end and stop, call provider sync-back hooks
- on resume, use persisted provider runtime state to reattach or restore the environment

Session should not learn Daytona or E2B details. It should only understand:

- which environment profile was selected
- what runtime state needs to be persisted for resume/cleanup
- when to ask for sync and teardown

#### Changes to `store` and `globaldb`

Files affected:

- `internal/store/types.go:474-507`
- `internal/store/globaldb/global_db.go:16-40`
- `internal/store/globaldb/global_db_workspace.go:15-184`

Changes:

- persist workspace-level environment selection
- persist session-level environment runtime state
- optionally index a subset of session environment fields in the global sessions table for filtering and inspection

#### Changes to `daemon`

Files affected:

- `internal/daemon/daemon.go:377-388`

Changes:

- wire the environment manager / local launcher / provider registry from the composition root

This is exactly where AGH wants provider construction to happen.

### 3.3 3c. Workspace alignment

#### Should a workspace optionally specify a backend?

Yes.

This should be a workspace concern, not an agent concern.

Reasoning:

- a workspace defines the filesystem and skill/config visibility model
- the execution environment changes filesystem semantics, sync requirements, and terminal execution location
- those are properties of the workspace context, not properties of an individual `AgentDef`

So the selection model should be:

1. `Workspace.EnvironmentRef` if explicitly set on the registered workspace
2. otherwise `Config.Defaults.Environment`
3. otherwise implicit `local`

This mirrors the existing default-agent override model:

- workspace row stores a local override (`internal/workspace/workspace.go:29-37`)
- `buildResolvedWorkspace(...)` loads config and applies workspace overrides (`internal/workspace/resolver.go:223-242`)

#### Should `ResolvedWorkspace` gain an environment field?

Yes:

```go
type ResolvedWorkspace struct {
	Workspace
	Config      aghconfig.Config
	Agents      []aghconfig.AgentDef
	Skills      []SkillPath
	Environment environment.Resolved
	ResolvedAt  time.Time
}
```

This is the right place because `ResolvedWorkspace` already means "runtime snapshot after applying config + discovery + overrides". Environment selection belongs in that same snapshot.

#### How should memory and skills behave?

Keep AGH's existing workspace identity local and canonical:

- `Workspace.RootDir` remains the canonical AGH workspace root
- config discovery remains local
- skill discovery remains local
- session metadata keeps referencing `WorkspaceID` and the local workspace identity (`internal/session/session.go:66-91`, `internal/store/types.go:474-507`)

Remote environments are projections of the workspace, not replacements for the workspace.

Implications:

- MCP servers remain daemon-local unless a profile explicitly asks otherwise
- startup prompt assembly remains daemon-local
- global/workspace skill resolution remains unchanged
- only the executable workspace tree is synchronized to the remote runtime

#### Should mixed mode exist?

Yes, absolutely.

Mixed mode is mandatory:

- some workspaces should stay `local`
- some can opt into `daytona`
- some can opt into `e2b`

The default must remain `local`. AGH is explicitly local-first today, and the current code assumes local discovery, local config, and local filesystem identity everywhere.

#### One subtle but critical point: local path vs runtime path

Today `Session.Workspace` is the local workspace root string (`internal/session/session.go:66-91`) and `acp.StartOpts.Cwd` is that same root (`internal/session/manager_start.go:189-199`).

For remote environments that is no longer safe to assume.

Example:

- local workspace root: `/home/pedronauck/project`
- Daytona runtime root: `/home/daytona/workspace`
- E2B runtime root: `/home/user/workspace`

If AGH keeps telling the agent the local path while the runtime path is different, ACP file and terminal behavior becomes confusing and sometimes wrong.

So the architecture needs two path concepts:

- canonical local workspace path
- runtime-visible workspace path

That is why `SessionEnvironmentState` should persist `RuntimeRootDir` separately instead of overwriting `Workspace.RootDir`.

### 3.4 3d. Concrete data model

#### Config model

I would extend config like this:

```go
// internal/config/config.go
type DefaultsConfig struct {
	Agent       string `toml:"agent"`
	Provider    string `toml:"provider,omitempty"`
	Environment string `toml:"environment,omitempty"`
}

type Config struct {
	Daemon        DaemonConfig
	HTTP          HTTPConfig
	Defaults      DefaultsConfig
	Limits        LimitsConfig
	Session       SessionConfig
	Permissions   PermissionsConfig
	MCPServers    []MCPServer
	Providers     map[string]ProviderConfig
	Environments  map[string]EnvironmentProfile `toml:"environments"`
	Observability ObservabilityConfig
	Log           LogConfig
	Memory        MemoryConfig
	Skills        SkillsConfig
	Extensions    ExtensionsConfig
	Automation    AutomationConfig
	Hooks         HooksConfig
	Network       NetworkConfig
}

type EnvironmentProfile struct {
	Backend       string           `toml:"backend"`
	SyncMode      string           `toml:"sync_mode,omitempty"`
	Persistence   string           `toml:"persistence,omitempty"`
	RuntimeRoot   string           `toml:"runtime_root,omitempty"`
	DestroyOnStop bool             `toml:"destroy_on_stop,omitempty"`
	Network       NetworkProfile   `toml:"network,omitempty"`
	Daytona       DaytonaProfile   `toml:"daytona,omitempty"`
	E2B           E2BProfile       `toml:"e2b,omitempty"`
}

type NetworkProfile struct {
	AllowPublicIngress bool     `toml:"allow_public_ingress,omitempty"`
	AllowOutbound      bool     `toml:"allow_outbound,omitempty"`
	AllowList          []string `toml:"allow_list,omitempty"`
	DenyList           []string `toml:"deny_list,omitempty"`
}

type DaytonaProfile struct {
	APIURL      string        `toml:"api_url,omitempty"`
	Target      string        `toml:"target,omitempty"`
	Image       string        `toml:"image,omitempty"`
	Class       string        `toml:"class,omitempty"`
	AutoStop    time.Duration `toml:"auto_stop,omitempty"`
	AutoArchive time.Duration `toml:"auto_archive,omitempty"`
}

type E2BProfile struct {
	APIURL             string        `toml:"api_url,omitempty"`
	Template           string        `toml:"template,omitempty"`
	Timeout            time.Duration `toml:"timeout,omitempty"`
	Secure             bool          `toml:"secure,omitempty"`
	AllowPublicTraffic bool          `toml:"allow_public_traffic,omitempty"`
}
```

TOML example:

```toml
[defaults]
agent = "general"
environment = "local"

[environments.local]
backend = "local"

[environments.daytona-dev]
backend = "daytona"
sync_mode = "turn-bidirectional"
persistence = "reuse"
runtime_root = "/home/daytona/workspace"
destroy_on_stop = false

[environments.daytona-dev.network]
allow_public_ingress = false
allow_outbound = true

[environments.daytona-dev.daytona]
api_url = "https://app.daytona.io/api"
target = "team-default"
image = "ubuntu:24.04"
class = "cpu-2"
auto_stop = "30m"
auto_archive = "24h"

[environments.e2b-ci]
backend = "e2b"
sync_mode = "session-bidirectional"
persistence = "transient"
runtime_root = "/home/user/workspace"
destroy_on_stop = true

[environments.e2b-ci.network]
allow_public_ingress = false
allow_outbound = true

[environments.e2b-ci.e2b]
api_url = "https://api.e2b.dev"
template = "base"
timeout = "1h"
secure = true
allow_public_traffic = false
```

#### Workspace model

I would extend the workspace registration model like this:

```go
// internal/workspace/workspace.go
type Workspace struct {
	ID             string
	RootDir        string
	AdditionalDirs []string
	Name           string
	DefaultAgent   string
	EnvironmentRef string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
```

And resolver options:

```go
// internal/workspace/resolver.go
type RegisterOptions struct {
	RootDir        string
	Name           string
	AdditionalDirs []string
	DefaultAgent   string
	EnvironmentRef string
}

type UpdateOptions struct {
	Name           *string
	AdditionalDirs *[]string
	DefaultAgent   *string
	EnvironmentRef *string
}
```

#### Session metadata model

I would extend session metadata like this:

```go
// internal/store/types.go
type SessionEnvironmentMeta struct {
	Backend               string    `json:"backend"`
	Profile               string    `json:"profile,omitempty"`
	InstanceID            string    `json:"instance_id,omitempty"`
	RuntimeRootDir        string    `json:"runtime_root_dir,omitempty"`
	RuntimeAdditionalDirs []string  `json:"runtime_additional_dirs,omitempty"`
	ResumeToken           string    `json:"resume_token,omitempty"`
	PreparedAt            time.Time `json:"prepared_at,omitempty"`
}

type SessionMeta struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name,omitempty"`
	AgentName    string                  `json:"agent_name"`
	WorkspaceID  string                  `json:"workspace_id,omitempty"`
	Channel      string                  `json:"channel,omitempty"`
	SessionType  string                  `json:"session_type,omitempty"`
	State        string                  `json:"state"`
	StopReason   *StopReason             `json:"stop_reason,omitempty"`
	StopDetail   string                  `json:"stop_detail,omitempty"`
	ACPSessionID *string                 `json:"acp_session_id,omitempty"`
	Environment  *SessionEnvironmentMeta `json:"environment,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}
```

#### Database schema changes

Workspace table:

```sql
ALTER TABLE workspaces ADD COLUMN environment_ref TEXT NOT NULL DEFAULT '';
```

Sessions table:

```sql
ALTER TABLE sessions ADD COLUMN environment_backend TEXT NOT NULL DEFAULT 'local';
ALTER TABLE sessions ADD COLUMN environment_profile TEXT NOT NULL DEFAULT 'local';
ALTER TABLE sessions ADD COLUMN environment_instance_id TEXT DEFAULT '';
```

Why split this way:

- workspace registration needs queryable environment selection
- session index should expose enough environment state for list/status/debug
- full provider resume/runtime state still belongs in `session.meta.json`

I would not put provider-specific blobs into the global sessions table. Keep detailed provider state in session metadata, not the index.

## 4. Risks and Open Questions

### 4.1 The biggest risk is not provisioning, it is filesystem authority

Provisioning a remote environment is the easy part.

The hard problem is: where is the authoritative workspace state?

Today the answer is obviously "local disk". That assumption is everywhere:

- workspace registration stores a local `RootDir` (`internal/workspace/workspace.go:29-37`)
- config loading reads local files (`internal/config/config.go:198-260`)
- scan/discovery walks local filesystem (`internal/workspace/scanner.go:37-157`)
- ACP file handlers read and write local files (`internal/acp/handlers.go:180-213`)

If the agent runs in Daytona or E2B and edits files through terminal commands, those changes are remote unless AGH syncs them back.

That means a real solution needs explicit sync behavior. Without it, remote execution is only half-integrated.

### 4.2 PTY is not automatically a safe ACP transport

ACP expects clean request/response framing over stdio.

Provider APIs often expose PTY and command abstractions, but PTY is a terminal stream, not necessarily a clean byte-oriented stdio channel. ANSI escapes, echo, and shell wrapping can corrupt ACP framing.

Open question:

- does each provider support a long-lived, bidirectional, non-PTY process stream suitable for ACP?

If not, AGH may need a small provider-side bridge process or a transport adapter.

This is the most important technical validation item before implementing the first remote provider.

### 4.3 Permission enforcement becomes split-brain if done naively

Current permission logic is daemon-local and root-path-based (`internal/acp/permission.go:94-181`).

In remote environments:

- ACP file requests can still be mediated by AGH through a provider-backed `ToolHost`
- but terminal commands can mutate files without passing through `fs/write`

So permission approval and filesystem sync need to be designed together. Otherwise AGH will approve a tool call but still lose track of terminal-induced file changes.

### 4.4 Resume semantics are provider-specific

AGH resume today is centered on ACP session reload plus session metadata repair (`internal/session/manager_lifecycle.go:37-99`, `internal/session/manager_start.go:72-99`).

With environments, resume may mean:

- reconnect to a running sandbox
- restart an archived sandbox
- recreate a transient sandbox and re-sync the workspace

That makes `SessionEnvironmentMeta` mandatory. `ACPSessionID` alone will not be enough anymore.

### 4.5 Dependency weight matters

AGH is a single-binary Go daemon. Do not solve this by shelling out to provider CLIs or bundling Node/Python helpers.

My recommendation:

- Daytona: use the official Go SDK, but keep it behind a thin internal provider adapter
- E2B: likely use direct HTTP/WebSocket clients from Go unless official Go docs or libraries appear and look stable

Do not let provider SDKs leak across packages. `daemon` should instantiate providers, and the rest of AGH should only see AGH-owned interfaces.

### 4.6 What AGH should explicitly not do

- Do not put `environment` or `sandbox` fields on `AgentDef`. That would mix model/tool choice with workspace topology.
- Do not overwrite `Workspace.RootDir` with a remote runtime path.
- Do not add Daytona/E2B branches directly inside `session.Manager`.
- Do not make `acp` import provider packages directly without an abstraction layer.
- Do not ship a first remote provider without an explicit sync contract.
- Do not try to implement Daytona and E2B simultaneously.

## 5. Recommended Phased Implementation Plan

### Phase 1: Land the abstraction with `local` only

Goal:

- no user-visible behavior change
- create the structural seams once, correctly

Work:

1. Add `Defaults.Environment`, `Config.Environments`, and validation.
2. Add `Workspace.EnvironmentRef` and `ResolvedWorkspace.Environment`.
3. Add `SessionEnvironmentMeta`.
4. Refactor `acp` to use injected `Launcher` and `ToolHost`.
5. Implement `local` as the first provider, reusing today's subprocess, permission policy, and terminal manager behavior.
6. Add tests proving the local provider preserves current create/prompt/stop/resume behavior.

Why this is the right first phase:

- it keeps all risk internal to AGH's architecture
- it avoids a half-baked provider implementation coupled to unstable seams
- it lets the team review the abstractions while behavior is still local and testable

### Phase 2: Ship Daytona as the first remote provider

Goal:

- first real remote execution environment
- workspace-aligned, persistent remote runtime

Work:

1. Implement `environment/daytona` behind the provider interface.
2. Support create/reuse/start/stop/archive/destroy flows.
3. Implement workspace sync:
   - sync local root + additional dirs to runtime before start
   - sync back on turn end and on stop
4. Persist Daytona sandbox identity in `SessionEnvironmentMeta`.
5. Support resume by reattaching to or restoring the same sandbox when possible.
6. Add a simple end-to-end profile for a workspace selected into Daytona.

Why Daytona first:

- official Go SDK
- stronger workspace/persistence fit
- lower friction for AGH's current local-workspace mental model

### Phase 3: Add E2B, but narrow the initial scope

Goal:

- support E2B without forcing it into the exact same usage shape as Daytona

Work:

1. Implement `environment/e2b`.
2. Start with explicit opt-in profiles only.
3. Prefer transient or automation-oriented use cases first.
4. Keep sync mode conservative until whole-workspace sync behavior is proven acceptable.

I would not make E2B the default remote provider for interactive local-repo coding on day one. Its docs suggest a better fit for fast transient sandboxes than for mirroring rich local workspaces.

### Phase 4: Harden sync, observability, and recovery

Goal:

- make remote execution production-worthy

Work:

1. Track sync durations and failures in observability.
2. Add environment info to status/list APIs.
3. Add leaked-environment cleanup on daemon restart.
4. Add crash-path sync attempts where safe.
5. Add explicit policies for secrets/env propagation and network restrictions.

## Final recommendation

If the team wants the shortest path that still respects AGH's architecture, do this:

1. Introduce `internal/environment/` and the `local` provider first.
2. Refactor `acp` to separate ACP protocol from launch/tool-host implementation.
3. Add workspace-level environment selection through config profiles plus a workspace override.
4. Ship Daytona first.
5. Add E2B second, likely starting with explicit automation-oriented profiles.

That path matches the existing code shape:

- `workspace` continues to own workspace identity and resolution
- `session` continues to own lifecycle orchestration
- `acp` continues to own ACP
- `daemon` remains the sole composition root

It also avoids the main design mistake here: pretending that "remote sandbox support" is just a different subprocess command. In AGH, it is not. It changes launch, filesystem, terminal execution, permissions, sync, and resume. The architecture has to model that explicitly.
