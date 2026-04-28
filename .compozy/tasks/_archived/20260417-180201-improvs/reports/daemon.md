# Improvements Report — internal/daemon

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmark plan and results table below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo $(rg --files internal/daemon -g '*.go' -g '!**/*_test.go' | sort) | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 16 | `appendCoreProjectorRegistrations` | `internal/daemon/daemon.go:703` |
| 16 | `(*resourceAgentCatalog).lookupAgent` | `internal/daemon/agent_skill_resources.go:192` |
| 16 | `(*hookBindingSourceSyncer).Sync` | `internal/daemon/hook_bindings.go:79` |
| 16 | `(*agentSkillSourceSyncer).Sync` | `internal/daemon/agent_skill_resources.go:386` |
| 15 | `(*Daemon).boot` | `internal/daemon/boot.go:128` |
| 14 | `extensionManifestToolMCPDeclarationProvider` | `internal/daemon/tool_mcp_resources.go:577` |
| 14 | `(*resourceAgentCatalog).agentsForWorkspace` | `internal/daemon/agent_skill_resources.go:295` |
| 13 | `verifyImportBoundaries` | `internal/daemon/boundary.go:60` |
| 13 | `acquireLock` | `internal/daemon/lock.go:57` |
| 12 | `extensionManifestBundleDeclarationProvider` | `internal/daemon/bundle_resources.go:264` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/daemon/bridges.go` | 1583 | Bridge CRUD, lifecycle locking, rollback, resource projection, reload coordination, and secret binding behavior are concentrated in one unit. |
| `internal/daemon/boot.go` | 1488 | Boot orchestration, dependency wiring, resource registration, service startup, and shutdown cleanup logic all live in one monolith. |
| `internal/daemon/hooks_bridge.go` | 1133 | Hook runtime adapters, fanout notifiers, dispatch helpers, and declaration providers are bundled together. |
| `internal/daemon/daemon.go` | 1111 | Core runtime state, option wiring, service factories, shutdown sequencing, and boundary helpers share one file. |
| `internal/daemon/agent_skill_resources.go` | 977 | Agent/skill resource syncers, catalog lookup, declaration providers, projectors, and cloning helpers are tightly packed. |
| `internal/daemon/tool_mcp_resources.go` | 731 | Shared resource catalogs, tool/MCP syncers, declaration providers, projectors, and clone helpers are co-located. |
| `internal/daemon/sandbox_reconcile.go` | 587 | Boot-time environment reattachment, metadata scanning, session recovery, and cleanup policy are combined in one unit. |
| `internal/daemon/bundle_resources.go` | 538 | Bundle publication, activation resources, and extension declaration harvesting share one file. |
| `internal/daemon/task_runtime.go` | 426 | Task session bridging, stop semantics, recovery bookkeeping, and helper mappings are bundled together. |
| `internal/daemon/hook_agent_events.go` | 337 | ACP agent-event normalization, JSON decoding, and permission/tool hook dispatch all live in one file. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/daemon | rg -v '_test.go'` notable production duplicates:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/daemon/agent_skill_resources.go:555-673` | `internal/daemon/tool_mcp_resources.go:359-484` | Managed resource sync loops for agent/skill/MCP and tool/MCP still perform the same list-by-source, compare, put, and stale-delete choreography. |
| `internal/daemon/agent_skill_resources.go:510-598` | `internal/daemon/agent_skill_resources.go:555-643` | `syncAgents` and `syncSkills` still repeat the same canonical snapshot replacement logic with only store/codecs changed. |
| `internal/daemon/agent_skill_resources.go:934-974` | `internal/daemon/tool_mcp_resources.go:649-689` | Validation/clone helper tails still repeat MCP/tool/agent helper patterns across resource publication files. |
| `internal/daemon/bundle_resources.go:104-126` | `internal/daemon/hook_bindings.go:43-65` | Bundle and hook publisher adapter helpers share the same nil-safe wrapper shape. |
| `internal/daemon/hooks_bridge.go:887-902` | `internal/daemon/hooks_bridge.go:904-919` | Config and agent declaration providers are near-identical wrappers over `workspaceHookDeclarations`. |
| `internal/daemon/bridges.go:1487-1508` | `internal/daemon/bridges.go:1510-1531` | Bridge lifecycle context lock helpers duplicate extension and instance lock bookkeeping. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*resourceCatalog[T]).Snapshot` | `internal/daemon/tool_mcp_resources.go:61` | Agent/tool/MCP resource-backed readers clone full record slices on every snapshot, so catalog cloning is an allocation-heavy shared path. | `BenchmarkResourceCatalogSnapshotAgentRecords` |
| `(*resourceAgentCatalog).ResolveAgent` | `internal/daemon/agent_skill_resources.go:172` | Session startup and resume resolve the effective agent through the resource-backed catalog for every workspace-bound session. | `BenchmarkResourceAgentCatalogResolveAgentWorkspaceHit` |
| `(*agentSkillSourceSyncer).Sync` | `internal/daemon/agent_skill_resources.go:329` | Boot and extension reload rebuild the managed agent/skill/MCP snapshot, often on no-op refreshes where comparison cost dominates. | `BenchmarkAgentSkillSourceSyncerSyncNoop` |
| `(*toolMCPSourceSyncer).Sync` | `internal/daemon/tool_mcp_resources.go:257` | Boot and extension reload also rebuild tool/MCP publication state, making the no-op comparison path the relevant steady-state benchmark. | `BenchmarkToolMCPSourceSyncerSyncNoop` |

### Optimization — Benchmark Results

Baseline command for `before` numbers: `go test -bench=. -benchmem -count=5 ./internal/daemon/...` before production fixes.
Final command for `after` numbers: `go test -bench=. -benchmem -count=5 ./internal/daemon/...` after all `internal/daemon/` changes landed.

Numbers below are the arithmetic mean of the five benchmark runs from each command.

| Benchmark | Before ns/op | Before B/op | Before allocs/op | After ns/op | After B/op | After allocs/op | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `BenchmarkResourceCatalogSnapshotAgentRecords` | 61,352 | 245,760 | 1,537 | 59,152 | 245,760 | 1,537 | fixed — `cloneResourceRecords` now preallocates the clone slice, trimming this shared snapshot path by 3.6% with no behavior change. |
| `BenchmarkResourceAgentCatalogResolveAgentWorkspaceHit` | 724,217 | 989,192 | 13,375 | 63,773 | 246,128 | 1,545 | fixed — `ResolveAgent` now scans the catalog once, preserves precedence, and falls back to the resolved workspace snapshot on a catalog miss; 91.2% faster and 75.1% less heap. |
| `BenchmarkAgentSkillSourceSyncerSyncNoop` | 420,991 | 373,322 | 6,142 | 420,752 | 373,328 | 6,142 | deferred — no package-local change produced a material steady-state win in the no-op sync path. |
| `BenchmarkToolMCPSourceSyncerSyncNoop` | 380,687 | 310,327 | 6,003 | 379,181 | 310,326 | 6,003 | deferred — no package-local change produced a material steady-state win in the no-op sync path. |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/daemon/boot.go:1399` | Skills watcher started by `startSkillsWatcher` | Derived `watcherCtx` cancellation plus `done` channel observed by `stopSkillsWatcher` | The goroutine only wraps `watcher.Start(watcherCtx)` and closes `done` on exit. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/daemon/daemon.go:286` | external | Injected signal bridge or OS signal source | External caller when injected; otherwise `signal.Stop` unregisters the local source | `Run` select loop | Receive-only field for shutdown signals. |
| `internal/daemon/daemon.go:291` | 0 | Daemon boot lifecycle | `publishBootState` closes it once readiness is published | Runtime/tests waiting for boot readiness | Constructed in `New` and closed exactly once behind `readyClosed`. |
| `internal/daemon/daemon.go:321` | 0 | Daemon shutdown lifecycle | `startSkillsWatcher` goroutine via `done` channel | `stopSkillsWatcher` during shutdown | Stored on the daemon after boot finalization. |
| `internal/daemon/boot.go:79` | 0 | Boot state handoff for skills watcher | `startSkillsWatcher` goroutine via `done` channel | `stopSkillsWatcher` through boot cleanup and daemon shutdown | Mirrors the daemon field before final publish. |
| `internal/daemon/boot.go:1399` | 0 | Skills watcher goroutine | Deferred `close(done)` in watcher goroutine | `stopSkillsWatcher` | Internal completion signal for watcher shutdown. |
| `internal/daemon/daemon.go:1076` | 1 | Local OS signal registration path | Not explicitly closed; `signal.Stop(ch)` unregisters notifications | `Run` select loop | Buffered one-slot channel created only when no injected signal bridge exists. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/daemon/daemon.go:265` | write-heavy | Daemon runtime state transitions during boot/detach/reset | Guards singleton boot/shutdown state and runtime pointers. |
| `internal/daemon/boot.go:67` | read-heavy | `bootState.extensions` handoff during extension boot/reload | Keeps extension-runtime snapshots synchronized while boot wiring mutates them. |
| `internal/daemon/bridges.go:60` | write-heavy | Bridge lifecycle lock maps | Serializes extension/instance lifecycle lock acquisition. |
| `internal/daemon/bridges.go:63` | read-heavy | Active extension runtime on the bridge runtime | Separates lifecycle lock bookkeeping from runtime pointer reads. |
| `internal/daemon/bridges.go:68` | write-heavy | Per-extension or per-instance lifecycle lock refcount | Refcounted lock object used by lifecycle context helpers. |
| `internal/daemon/hooks_bridge.go:131` | read-heavy | Session lifecycle observer slice | Snapshot-based fanout avoids holding the lock while invoking observers. |
| `internal/daemon/hooks_bridge.go:174` | read-heavy | Hook telemetry sink slice | Snapshot-based fanout avoids holding the lock while writing hook telemetry. |
| `internal/daemon/hooks_bridge.go:219` | read-heavy | Hook runtime pointer and agent-event notifier pointer | Guards notifier reconfiguration while dispatchers take snapshots. |
| `internal/daemon/tool_mcp_resources.go:40` | read-heavy | Generic resource catalog revision + record snapshot | Shared by agent/tool/MCP catalog projections and lookups. |

### Concurrency — Select Audit

All production `select` statements under `internal/daemon/` include `ctx.Done()` or are bounded by a caller-provided signal/input source:

- `internal/daemon/orphan.go:64` waits on `ctx.Done()`, ticker, and grace timer while polling orphan exit.
- `internal/daemon/daemon.go:917` waits on daemon context cancellation or the shutdown signal source.

### Security — Threat Model

- Trust boundaries:
  - Local UDS/HTTP API callers cross into daemon-managed extension and bridge lifecycle operations.
  - Agent subprocess events cross into hook dispatch through `dispatchACPAgentHookEvent`.
  - Process environment and AGH home paths cross into secret resolution, lock/info files, and daemon startup metadata.
- Attacker capabilities:
  - A local caller can submit extension install requests and bridge create/update requests through daemon APIs.
  - A bridge or extension configuration author can control provider config bytes, routing defaults, display names, and secret binding references that reach this package.
  - An agent subprocess can emit ACP tool/permission events with arbitrary raw JSON payloads.
  - The attacker does not control operator-owned AGH home paths or process environment unless they already have the daemon’s local execution privileges.
- In-scope assets:
  - Integrity of bridge instance state and extension install metadata.
  - Correct scoping of bridge secret environment lookups.
  - Stability of hook dispatch when processing agent-event payloads.
  - Integrity of daemon readiness, lock, and info state under local runtime management.
- Out-of-scope:
  - Full trust of installed extension code after it leaves this package and begins executing.
  - OS-level compromise or malicious operator control over `AGH_HOME` / process environment.
  - Downstream execution performed by other packages after validated requests leave the daemon composition root.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/daemon/bridges.go:140-274` | API-facing `bridgepkg.CreateInstanceRequest` / `UpdateInstanceRequest` payloads | `BridgeInstanceSpecFromCreateRequest`, `req.Validate`, scope normalization, lifecycle locking, and resource-store versioning | Bridge service/resource store writes and extension reload coordination | LOW — request validation and resource projection fail closed before any daemon-owned side effects beyond canonical state writes. |
| `internal/daemon/bridge_secrets.go:35-61` | Persisted bridge secret bindings (`binding.VaultRef`) | `parseEnvBridgeSecretRef` trims, enforces `env:NAME`, and validates the env-var name against a strict regex | `getenv(envName)` returns secret material to the bridge runtime | LOW — env references are constrained to explicit, validated names and reject missing/empty values. |
| `internal/daemon/extensions.go:87-100` | API-facing `contract.InstallExtensionRequest.Path` and checksum | `strings.TrimSpace` for manifest load plus extension-package manifest/checksum validation in `InstallLocalManaged` | Local filesystem extension install into managed home paths | LOW — this path is local-operator scoped and relies on manifest/checksum validation before managed install. |
| `internal/daemon/hook_agent_events.go:35-64`, `203-224` | Agent subprocess ACP event type + raw JSON payload | Event type gate, `normalizeHookAgentEvent`, JSON decode failure returns false, field trimming, and payload cloning before dispatch | Hook runtime dispatch for tool and permission events | LOW — malformed payloads are dropped and the package does not execute user-provided code or templates here. |
| `internal/daemon/daemon.go:1071-1079`, `internal/daemon/orphan.go:25-107`, `internal/daemon/info.go:60-129`, `internal/daemon/lock.go:50-132` | Daemon home paths, process table, and OS signal/input state | Paths come from resolved home config; PID/path validation and bounded OS command usage protect runtime invariants | Lock/info file management, signal subscription, orphan cleanup | REJECTED — these are operator/system-controlled surfaces, not lower-trust attacker input in this threat model. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `DAEMON-OPT-001` | extreme-software-optimization | medium | `internal/daemon/agent_skill_resources.go:172-245` | `ResolveAgent` rebuilt and sorted the full workspace/global agent projection for every lookup and ignored the resolved-workspace snapshot whenever a catalog existed but missed the requested agent. | fixed — replaced the full projection walk with a single catalog scan that preserves precedence and falls back to `resolved.Agents` on miss; regression test added and benchmark improved 91.2%. |
| `DAEMON-OPT-002` | extreme-software-optimization | low | `internal/daemon/tool_mcp_resources.go:703-712` | `cloneResourceRecords` used append-based growth for fixed-length copies in the shared resource snapshot path. | fixed — switched to fixed-length preallocation and index assignment; benchmark improved 3.6% with identical semantics. |
| `DAEMON-REF-001` | refactoring-analysis | low | `internal/daemon/agent_skill_resources.go:555-673`, `internal/daemon/tool_mcp_resources.go:359-484` | Agent/skill and tool/MCP managed-resource syncers still duplicate the same list-by-source, compare, put, and stale-delete choreography. | deferred — a safe dedup would require a shared generic sync abstraction across multiple projector pipelines, which is outside this package-local improvement pass. |
| `DAEMON-REF-002` | refactoring-analysis | low | `internal/daemon/boot.go:128`, `internal/daemon/bridges.go:140-274` | Core boot orchestration and bridge lifecycle logic remain concentrated in >1k LOC files with mixed responsibilities. | deferred — decomposition is desirable but spans package-wide boot wiring and bridge lifecycle contracts, with no isolated correctness or security bug to justify a rushed split in this pass. |

## Per-Skill Notes

### refactoring-analysis

- The refreshed cyclomatic inventory shows `lookupAgent` immediately enters the top-10 after the resolver fix, which is acceptable for this pass because the added branching replaces a far more expensive full projection sort on every lookup.
- The two refactoring findings carried forward are structural duplicates in the managed resource syncers and the large-file concentration in `boot.go` / `bridges.go`. Both are deferred because they require broader package re-slicing rather than a targeted correctness or security fix.

### extreme-software-optimization

- Benchmarks were added for all four hot-path candidates before production changes.
- Two performance findings met the evidence bar and were fixed: `ResolveAgent` and `cloneResourceRecords`.
- The no-op syncer benchmarks were rerun after the fixes and remained effectively flat, so no speculative optimization was landed there.

### ubs

- `not-run` due missing skill-runner interface in this session; no manual substitute is being used.

### deadlock-finder-and-fixer

- Inventory review found one daemon-owned goroutine (`startSkillsWatcher`), a small channel set, and cancellation-aware `select` sites.
- No deadlock, leaked-goroutine, or lock-ordering issue was identified inside `internal/daemon/` during this pass.

### security-review

- The threat model and attacker-input surface inventory were completed before the final verdict.
- No HIGH or MEDIUM security issue was identified in-package. The exposed surfaces are either validation-gated request payloads, strictly parsed env-secret references, or operator-owned runtime paths that are outside the lower-trust attacker model for this task.

## Deferred Items (carry forward)

- Consider extracting the shared managed-resource sync choreography from `agent_skill_resources.go` and `tool_mcp_resources.go` into a common package-local helper after this improvements wave settles.
- Plan a dedicated follow-up split for `boot.go` and `bridges.go` so boot orchestration and bridge lifecycle logic can be decomposed without mixing it into an optimization pass.

## `make verify`

Final command: `make verify`

Excerpt:

```text
0 issues.
✓  internal/daemon (8.71s)
✓  internal/cli (9.456s)

DONE 4463 tests in 11.029s
OK: all package boundaries respected
```
