# Hermes vs AGH — Tools, Skills & Security

## Executive Summary

- **Largest release-blocking area by far.** Hermes ships ~9.8K LOC of guardrails (`approval.py` 994, `skills_guard.py` 928, `process_registry.py` 1205, `tirith_security.py` 684, `mcp_oauth*.py` 1100+). AGH ships ~150 LOC equivalent: 3-mode `PermissionMode` (`internal/acp/permission.go`), 11 regexes in `skills/verify.go`, SHA-256 directory hash (`skills/provenance.go`). No dangerous-command detection, no SSRF guard, no website blocklist, no OSV check, no skill content scanner, no MCP OAuth, no process registry, no tool interrupt.
- **No dangerous-command approval flow.** `acp/permission.go` encodes only `deny-all | approve-reads | approve-all` and delegates choice to the ACP SDK (`selectPermissionOutcome`). Hermes combines 44+ regexes, Tirith content scanning, Smart Approval via aux-LLM, four-way `once / session / always / deny` persistence, and cosign-verified auto-install.
- **No skill supply-chain defense.** AGH hashes directories and records a sidecar (`skills/provenance.go:47`) but never *scans* content. Hermes `skills_guard.py` runs 100+ threat patterns across 12 categories and decides install by **trust-tier policy** (`builtin|trusted|community|agent-created`). Any `agh skills install` is an unbounded execution vector for an agent-created skill.
- **No SSRF / website policy.** Zero URL guards in AGH. Hermes `url_safety.py` blocks private/loopback/CGNAT/169.254 metadata, fails closed on DNS error; `website_policy.py` adds a user blocklist with wildcard + shared-file support and TTL cache. Any URL-capable MCP or extension in AGH can probe internal IPs or cloud metadata.
- **No process registry or tool interrupt.** AGH's `internal/hooks/executor_subprocess_unix.go` kills a single `exec.Cmd`; there is no global inventory, crash-checkpoint recovery, session-scoped `kill_all`, stdin `write/close`, PTY support, watch-pattern notifications, or per-thread interrupt.

## Capability-by-Capability Gap Analysis

### Permission model

- **Hermes**: per-session approval keys (`approval.py:207`), permanent allowlist in `config.yaml:command_allowlist` (`:395`), per-session YOLO (`:208`), alias table for legacy keys (`:147-162`), context-var session identity so parallel subagent threads don't poison each other (`:27-55`), dedicated cron mode (`:535-545`). ACP bridge (`acp_adapter/permissions.py:44`) maps the four `PermissionOptionKind` values.
- **AGH**: 3-mode policy (`config/config.go:77-85`). Matcher (`acp/permission.go:106-117`) ignores tool name/content/caller — only `acpsdk.ToolKindRead`. `ApproveAll` is the **bootstrap default** (`config/bootstrap.go:66`). Pending requests in a map keyed by `request_id`, 5-min timeout (`acp/permission.go:452-457`). No session/permanent allowlist, no YOLO, no per-pattern approvals, no aux-LLM.
- **Gap**: BLOCKER. With `approve-all` default and no dangerous-command detection, an untrusted ACP agent can `rm -rf /` and AGH confirms it.

### Approval flows

- **Hermes** `check_all_command_guards` (`approval.py:715`) orchestrates: Tirith scan → regex detection → aux-LLM smart-approval → CLI `[o]nce|[s]ession|[a]lways|[d]eny` → gateway queue-blocking with 300s timeout + 1s heartbeat slice for inactivity watchdog (`:882-911`). Each concurrent subagent gets its own `_ApprovalEntry` (`:220-230`).
- **AGH**: `AgentProcess.ResolvePermission` (`acp/permission.go:429`) just unblocks the channel. No detection, no smart-approval, no `always` persistence, no queue dedup.

### Path sandboxing

- **Hermes**: centralized `validate_within_dir` (`path_security.py:15-43`) used across skill manager, cronjob, credentials, skills-hub. Symlink-escape check on every scanned skill file (`skills_guard.py:755-767`).
- **AGH**: `acp/permission.go:139-234` resolves to the workspace root via `filepath.EvalSymlinks`, handles non-existent ancestors, rejects with `ErrPathOutsideWorkspace`. **ACP-only**. Skills loader (`skills/loader.go:165-207`) and provenance hasher (`skills/provenance.go:60-82`, `:203-221`) don't verify symlink targets stay inside the skill dir.
- **Gap**: HIGH. A skill directory symlinked to `~/.ssh/id_rsa` would be hashed by `ComputeDirectoryHash` — `writeHashEntry` even stores the target as metadata.

### URL safety / SSRF

- **Hermes** (`url_safety.py`): blocks `is_private|is_loopback|is_link_local|is_reserved|is_multicast|is_unspecified`, explicit CGNAT range `100.64.0.0/10`, explicit `metadata.google.internal` hostname block, DNS-fails-closed, narrow trusted-private-IP allowlist for QQ. Covers both IPv4 and IPv6 via `getaddrinfo` with AF_UNSPEC.
- **AGH**: none. The only `User-Agent` setter is `cli/client.go:1536` for the daemon's own HTTP client.
- **Gap**: BLOCKER once AGH exposes any URL-capable tool or MCP server to agents.

### Website policy

- **Hermes** (`website_policy.py`): YAML-driven blocklist with wildcard patterns, shared list files, 30s TTL cache, fail-open on config errors (but fail-closed when a test passes an explicit path).
- **AGH**: none.

### OSV / dependency malware check

- **Hermes** (`osv_check.py`): queries `api.osv.dev` for `MAL-*` advisories before spawning any npx/uvx MCP server. Fail-open on network error (`:40-62`).
- **AGH**: none. `config.MCPServer{Command, Args}` is spawned without vetting. Any `npx pkg-with-malware` is executed with AGH's privileges.

### Tirith content scanning & supply-chain verification

- **Hermes**: auto-installs the Tirith binary from GitHub releases with cosign keyless verification (`tirith_security.py:216-253`, pinning the OIDC workflow identity regex `:44`) plus SHA-256 checksum (`:256-278`). Persists failure reason with TTL to avoid retry storms (`:152-166`). Background install thread never blocks startup (`:524-603`). Exit-code-as-truth mapping (`0|1|2`), fail-open/fail-closed configurable.
- **AGH**: none. There is no content scanner and no signed-binary bootstrap story.

### MCP server management & OAuth

- **Hermes** (`mcp_oauth.py` + `mcp_oauth_manager.py` ~1130 LOC): OAuth 2.1 + PKCE, `HermesTokenStorage` with `0o600` + atomic writes (`mcp_oauth.py:158-168`), absolute `expires_at` persistence (`:244-253`), ephemeral localhost callback server, per-server token files, disk-mtime detection for external refreshes (CC-1096), 401 dedup via in-flight futures (`mcp_oauth_manager.py:76`), AS metadata prefetch for providers whose token endpoint differs from the MCP origin.
- **AGH**: MCP is declarative only (`config.MCPServer{Command,Args,Env}`, `skills/mcp.go`). No OAuth, no token storage, no scope gating. `skills/mcp.go:133-151` gates marketplace skills by a `Slug|Registry:Slug|Hash` allowlist; non-marketplace skills pass unconditionally (`:139`).
- **Gap**: HIGH for OAuth-backed MCP (Linear, Google Drive, Notion).

### Skill signing / integrity

- **Hermes**: content hash (`skills_guard.py:715-727`) + trust tiers + install-policy matrix (`:41-47`) + scan-verdict-driven `should_allow_install`.
- **AGH** (`skills/provenance.go`): SHA-256 of directory payload, sidecar JSON `{hash,registry,slug,version,installed_at}`, `VerifyHash` on reload. **No trust tiers, no scan verdict, no install-decision matrix** — `VerifyHash` only detects post-install tampering.
- **Gap**: HIGH. AGH will install a marketplace skill containing `rm -rf /` in `SKILL.md` as long as the hash matches. `skills/verify.go`'s 11 patterns run on list-time content verification — never gate install, never block use.

### Skill sync

- **Hermes** (`skills_sync.py`): manifest `skill_name:origin_hash` tracks bundled-vs-user-modified, auto-migrates v1→v2, respects user deletions, restores user copies on failed update via `.bak` swap, atomic write via `mkstemp`+`os.replace`.
- **AGH**: bundled skills come from `go:embed` (`skills/bundled/embed.go`), user skills live under `~/.agh/skills`, no manifest, no bundled-vs-customized detection, no restore path.

### Skills guard / manifest validation

- **Hermes** (`skills_guard.py`): 80+ regex patterns across 10 categories (exfiltration, injection, destructive, persistence, network, obfuscation, execution, traversal, mining, supply_chain, privilege_escalation, credential_exposure), invisible-unicode scan (zero-width, BOM, RTL-override), structural checks (file count `50`, total size `1MB`, single-file `256KB`, binary-extension denylist, executable-bit-on-non-script), symlink-escape.
- **AGH** (`skills/verify.go`): 11 regex patterns for prompt-injection + one content-length warning. No structural checks, no credential detection, no obfuscation detection, no invisible-unicode detection. `allowedFrontmatterFields` allows only `name|description|version|metadata` which is good but narrow.

### Tool budget / rate limiting

- **Hermes** `budget_config.py` + per-tool `max_result_size_chars` in `registry.py:97` + watch-pattern rate limit `WATCH_MAX_PER_WINDOW=8/10s` with overload kill switch (`process_registry.py:61-64`, `:187-220`).
- **AGH**: none at tool level. `internal/task/limits.go` exists for task orchestration but is not wired to tool invocations.

### Credential isolation

- **Hermes**: `credential_files.py` + `env_passthrough.py`; token writes use `0o600` (`mcp_oauth.py:164`). `_sanitize_subprocess_env` is used when spawning background processes (`process_registry.py:44`, `:345`, `:386`).
- **AGH**: MCP server env is passed through `config.MCPServer.Env` verbatim to the subprocess. Hooks pass `Env map[string]string` to subprocess commands (`hooks/executor_subprocess_unix.go`). No env scrubber, no file-perm enforcement on token caches.

### Tool interrupt

- **Hermes** (`interrupt.py`): per-thread `_interrupted_threads` set with ident keys so `Ctrl+C` in one gateway session never kills another session's tool. Tools poll `is_interrupted()` in hot loops. Backwards-compat proxy emulates `threading.Event`.
- **AGH**: `context.Context` cancellation on the hook dispatch path and ACP session. No per-thread/per-turn flag that long-running tools can poll without holding the cancellation channel — most MCP tools won't receive it.

### Process registry

- **Hermes** `process_registry.py`: tracked, LRU-pruned, checkpoint-on-write to `$HERMES_HOME/processes.json`, recovered across restart via `_is_host_pid_alive`, stdin `write|submit|close`, PTY mode via `ptyprocess`, watch patterns with rate limit, completion queue for agent resume, `kill_all(task_id)` on session reset.
- **AGH**: no equivalent. Subprocess lifecycle lives in `hooks/executor_subprocess.go` and `acp/*` per-invocation and is lost on daemon restart.

### Input validation / injection prevention

- **Hermes**: `tools/credential_files.py`, `tools/file_operations.py` + dedicated `tests/test_sql_injection.py`. Dangerous-command regex catches `DROP TABLE|TRUNCATE|DELETE FROM (no WHERE)` (`approval.py:87-89`).
- **AGH**: SQL is behind the sqlc layer; shell command injection surface is whatever the ACP agent does. No pre-flight command inspection.

## Patterns worth stealing

1. **`check_all_command_guards` orchestrator** (`approval.py:715`). Single function runs scanners → deduplicates findings → issues one combined approval prompt → persists decisions per-finding. AGH should have a `SessionGuard` invoked before every write/execute tool call.
2. **Trust-tier × verdict install matrix** (`skills_guard.py:41-47`). Keyed by source (`builtin|trusted|community|agent-created`) × verdict (`safe|caution|dangerous`). Clear, testable, auditable. AGH should replace `AllowedMarketplaceMCP` with this.
3. **Blocking gateway approval queue** (`approval.py:220-285`). Each approval gets its own `threading.Event` (AGH equivalent: channel) so parallel subagents block independently; `resolve_all` flag for `/approve all`.
4. **Cosign + SHA-256 binary bootstrap with failure-reason TTL marker** (`tirith_security.py:216-253`, `:152-166`). Retryable failure classes (`cosign_missing`) auto-clear when cause resolves.
5. **Per-thread interrupt** (`interrupt.py`). The cleanest way to interrupt only the owning session's tools in a multi-session daemon without threading `ctx` through every callback.
6. **Checkpoint-on-write + PID liveness probe on boot** (`process_registry.py:1003-1112`). AGH's hook subprocess path would benefit — today a daemon crash with live hooks leaves orphan processes.
7. **Disk-mtime-based cross-process token invalidation** (`mcp_oauth_manager.py:60-75`). Essential if the daemon grows multi-process or cooperates with a CLI refresh path.
8. **Unicode/ANSI normalization before regex matching** (`approval.py:169-184`). Strips `NFKC` + ANSI + nulls so obfuscation (`＝r＝m` fullwidth, `\x1b[...]` wrapping) can't bypass detection.
9. **Smart Approval via auxiliary LLM** (`approval.py:548-593`). Reduces false-positive friction without dropping detection.
10. **Website blocklist with TTL cache + wildcards** (`website_policy.py:131-199`). The 30s cache is the right knob — config changes visible inside a minute without per-call YAML parse.

## Explicitly skip

- **Feishu / Discord / HomeAssistant / Voice tools** and the large vendor-tool catalog in Hermes — out of AGH's single-binary scope.
- **Nous-branded managed tool gateway** (`managed_tool_gateway.py`) — vendor-specific OAuth bridge; AGH should keep MCP OAuth implementation-neutral.
- **Hermes `INSTALL_POLICY["agent-created"] = ("allow","allow","ask")`** — AGH should not let an agent self-create a skill and install it without human gate. Start stricter and relax later.
- **Tirith remote binary download** — supply-chain wise, prefer bundling a Go-native scanner (even a minimal one) over downloading a binary from GitHub Releases at first run. The cosign/SHA-256 pattern is still worth stealing for any future binary deps.
- **Legacy Python `_interrupt_event` proxy** (`interrupt.py:81-98`) — pure Python-artifact; Go's `context.Context` + a per-session `atomic.Bool` does the same job cleanly.

Key AGH file refs the remediation will touch: `/Users/pedronauck/Dev/compozy/agh/internal/acp/permission.go` (expand beyond 3 modes), `/Users/pedronauck/Dev/compozy/agh/internal/skills/verify.go` (expand patterns + add structural checks + wire into install), `/Users/pedronauck/Dev/compozy/agh/internal/skills/provenance.go` (add trust-tier resolver + install decision), `/Users/pedronauck/Dev/compozy/agh/internal/config/config.go:77-85` (add per-pattern allowlist + session/permanent scopes), `/Users/pedronauck/Dev/compozy/agh/internal/skills/mcp.go:133-151` (generalise marketplace allowlist), plus new packages for URL safety, website policy, OSV check, and process registry.
