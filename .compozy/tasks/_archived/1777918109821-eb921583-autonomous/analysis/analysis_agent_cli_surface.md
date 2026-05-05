# Agent CLI Surface — Gap Analysis for Truly Autonomous Agents

> Slice owner: the CLI surface that **AGENTS (running inside ACP subprocesses)** use to talk back to the AGH daemon.
> Audience: AGH core devs planning Phase 2/3 (Memory/Skills/State + Network).

---

## 1. TL;DR

AGH already exposes a rich operator CLI (`agh network ...`, `agh task ...`, `agh memory ...`) and the network manager even injects ready-to-paste `agh network send` examples into delivered envelopes
(`internal/network/delivery.go:820-988`). But the surface is **shaped for human operators**, not for agents:

1. **Agent identity is invisible to the daemon.** `agh whoami` only echoes env vars; nothing on the daemon side knows "the caller is `sess_X` running as agent `reviewer`". Every command therefore demands explicit `--session sess_X` and a fresh trip through `clientFromDeps()`.
2. **The two primitives an autonomous agent needs most — `claim a task` and `wait for the next message` — are either bolted on or missing.** `agh task run claim` exists but cannot be filtered by capability/queue. There is no `agh network recv --wait`, no `agh network whois <peer>`, no `agh network discover --capability task.write`, no `agh task next` / `agh task pull`.
3. **No agent-spawn primitive on the CLI.** Sub-agent creation requires `agh session new` (human ergonomics, expects `--workspace`, `--cwd`, etc.) and there is no way for an agent to spawn a worker sub-session bound to its current task/channel.
4. **Output is operator-friendly TOON/human, not a stable agent contract.** JSON exists per command but error codes are uniform `error: ...` text on stderr; no `--id`, no exit-code taxonomy, no idempotency keys for envelopes the way `task run` already provides.
5. **No `agh agent ...` introspection beyond definitions.** `agh agent list/info` shows static config, not the **live agents currently visible on the network** (peers + their capability briefs). A truly autonomous agent has no single command answering "who can do X right now?".

The biggest single gap: there is **no agent-scoped CLI verb set** — every command lives in a domain noun (`network`, `task`, `memory`) and forces the agent to plumb its own identity. The proposal in §4 introduces an `agh me ...` short surface plus capability discovery and blocking inbox primitives.

---

## 2. Current AGH CLI surface

Source of truth: `/Users/pedronauck/Dev/compozy/agh/internal/cli/root.go` plus the per-domain files. Command tree as of `main`:

| Top-level command | Subcommands (selected) | Today's caller | Agent-friendly? | Gap for autonomy |
|---|---|---|---|---|
| `agh daemon` | `start/stop/status` | human | n/a (daemon control) | — |
| `agh whoami` | (env echo only — `internal/cli/whoami.go`) | both | partial | doesn't ask the daemon "what role am I, what channel(s) am I in, what task am I working?" |
| `agh agent` | `list`, `info <name>` | human | static only | no live agent / peer roster; no `agh agent ps` for live ACP processes the daemon owns |
| `agh session` | `new`, `list`, `stop`, `status`, `resume`, `wait`, `prompt`, `events`, `history` | human | partial | no `--bind-task <id>`, no `--parent-session $AGH_SESSION_ID`, no `agh session spawn-child` |
| `agh network` | `status`, `peers [channel]`, `channels`, `send`, `inbox` | both | partial — agents can send | **no `recv --wait`, no `join`, no `leave`, no `whois`, no `discover --capability`, no `subscribe --kind`** |
| `agh task` | `list/create/get/update/cancel`, `child create`, `dependency add/remove`, `run list/enqueue/claim/start/attach-session/complete/fail/cancel` | both (network ingress already authenticates peers via `network/tasks.go`) | mostly | claim is per-run, not per-pool. No `agh task next --capability $cap` (poll). No `agh task lease-extend`. No agent-side ack of completion that auto-records `trace`. |
| `agh memory` | `health/history/list/search/read/write/delete/reindex/consolidate` | both | yes | no per-agent default scope; agents must compute workspace path themselves. No `--ttl`, no `--tag` for cross-agent retrieval. |
| `agh skill` | `list/view/info/create/install/remove/update/search` | both | yes | already used to inflate agent context; OK |
| `agh extension` | `list/install/enable/disable/<name>` | human | n/a | extensions are MCP-side, not agent-side |
| `agh hooks` | `hook-event` (hidden), `catalog/runs/events` | daemon-internal | n/a (hooks fire via NATS, not CLI) | — |
| `agh observe` | `events`, `health` | human | partial | events stream is operator-shaped; agents cannot subscribe to *their own* event tail with an idiomatic verb |
| `agh bridge` | inbound chat bridges | human | n/a | — |
| `agh automation` | jobs/triggers/runs | human | n/a (operator domain) | — |
| `agh workspace` | crud + resolve | human | partial | agents shouldn't need to manage workspaces; they need read-only resolution |
| `agh mcp` | auth | human | n/a | — |
| `agh config` / `install` / `update` / `uninstall` | bootstrap | human | n/a | — |

**Implicit CLI ↔ daemon contract:** every command opens a fresh UDS HTTP client (`clientFromDeps`, `internal/cli/client.go:443`). There is no persistent agent connection, no auth token tied to the caller, no per-call attribution. The daemon trusts the socket itself and infers nothing about the caller.

**Identity injection today** (`internal/cli/whoami.go:1-44`):

```go
const (
    envSessionID = "AGH_SESSION_ID"
    envAgentID   = "AGH_AGENT"
    envAgentName = "AGH_AGENT_NAME"
)
```

ACP launchers also set workspace + socket env vars, but the CLI **does not auto-resolve them** — every command still requires `--session`, `--channel`, `--workspace`, `--scope` flags. The most concrete proof this hurts agents is `internal/network/delivery.go:865-988`, where the manager hand-builds multi-line shell snippets repeating `--session "$AGH_SESSION_ID"`, `--channel "..."`, `--to "..."`, `--reply-to "..."`, `--causation-id "..."`, `--trace-id "..."`, `--interaction-id "..."` for every reply guidance. That entire string-construction path exists because the CLI does not infer caller identity. It is a glaring tell.

---

## 3. Reference comparisons

### 3.1 Hermes (`/Users/pedronauck/Dev/compozy/agh/.resources/hermes`)

- Hermes is a **single-agent REPL**, not a multi-agent platform — its "CLI" is mostly slash commands inside an interactive prompt loop (`hermes_cli/commands.py:61-180`).
- Useful inverse signal: Hermes exposes everything an agent might want (model swap, queue prompt, snapshot, retry, undo, branch, btw, steer, agents-list) as **stable, single-word slash verbs**. AGH is much more verb-heavy (`agh task run claim <id>`).
- Lesson: an autonomous agent benefits from **flat, low-arity verbs**: `recv`, `send`, `claim`, `pass`, `done`, `fail`, `whois` — not deeply nested noun trees.

### 3.2 Multica (`/Users/pedronauck/Dev/compozy/agh/.resources/multica/CLI_AND_DAEMON.md`)

This is the closest analogue to where AGH is heading:

- Daemon polls a server for tasks, **claims** them, runs an ACP-style agent CLI (Claude Code, Codex, OpenCode, OpenClaw, Hermes, Gemini, Pi, Cursor — the same matrix AGH targets), heartbeats every 15 s.
- Agent-callable surface includes `multica issue list/get/create/update/assign/status`, `issue comment add`, `issue runs`, `issue subscriber add` — **first-class collaboration verbs** that AGH lacks an equivalent for (no `agh task comment`, no `agh task subscribe`, no `agh task assign --to <agent>`).
- Polling cadence is **explicitly tunable** (`--poll-interval 3s`, `--heartbeat-interval 15s`). AGH's network manager owns delivery internally but offers no client-side blocking-pull primitive equivalent to "poll my queue every 3 s and give me the next task".
- Output is `--output table|json` consistently. Every list verb supports JSON. AGH already does this via `-o {human,json,toon}`, which is good — **keep that.**

### 3.3 OpenClaw (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/src/cli/`)

- Massive CLI surface (~150 files in `src/cli/`), but its "channels" are **chat-platform bindings** (Slack/Telegram/WhatsApp), not agent-mesh channels. Not directly applicable.
- The relevant inversion is `capability-cli.ts` and `pairing-cli.ts`: even OpenClaw, which is operator-heavy, exposes **capability discovery and pairing** as first-class CLI verbs. AGH has `network capability_brief` in code (`internal/network/capability_brief.go`) and ships catalogs in payloads (`NetworkCapabilityCatalogPayload`), but **nothing on the CLI** asks "show me what `peer-X` can do" or "find peers that have `task.write`".

### 3.4 Paperclip MCP server (`/Users/pedronauck/Dev/compozy/agh/.resources/paperclip/packages/mcp-server/README.md`)

This is the cleanest model of "what an agent's tool surface should look like":

- Every operation an agent can perform is a single MCP tool with a verb-style name: `paperclipCheckoutIssue`, `paperclipReleaseIssue`, `paperclipAddComment`, `paperclipAskUserQuestions`, `paperclipRequestConfirmation`, `paperclipApprovalDecision`.
- Two affordances stand out:
  1. `paperclipApiRequest` — an **escape hatch** that lets the agent call any REST endpoint when a tool isn't enough.
  2. `paperclipGetHeartbeatContext` — a **single read** that returns "everything an agent needs to know right now to make a decision" (current goals, assigned issues, pending approvals, running services).
- AGH's analogue today is "spawn an `agh whoami` + `agh task list --owner-ref $AGH_AGENT` + `agh network inbox --session $AGH_SESSION_ID`" — three round-trips. We need a `agh me context` (or `agh inbox`) that returns one consolidated payload.

### 3.5 Claude Code (inverse model)

Claude Code exposes Bash and a small set of typed tools to the LLM. The model **does not** parse JSON exit codes — it reads pretty text. Every Claude-Code-style tool has a `userFacingName`, a `description()` callback, a `prompt()`, and a `searchHint`. The CLI an agent calls can be either a Bash command or a typed MCP tool, but **either way the agent reads natural-language responses**.

Implication for AGH: the CLI must remain agent-readable in `human` mode (don't strip context to be "cleaner") AND machine-parseable in `json` mode. Today's `-o human` is good: it already includes the actionable hints (`See "agh network --help" for options.`). Keep that, and add a `-o jsonl` mode for streaming inboxes.

---

## 4. The proposed agent CLI

**Design principles:**

- **Identity is implicit.** Any command run with `AGH_SESSION_ID`/`AGH_AGENT` set picks them up — no `--session` flag required. Explicit flags still win for operator use.
- **Verbs over nouns.** Agent-facing surface is short verbs: `recv`, `send`, `claim`, `pass`, `done`, `fail`, `whois`, `discover`, `inbox`, `next`.
- **Two output modes for agents:** `-o json` (single object) and `-o jsonl` (event stream, one JSON per line, for `recv --wait`).
- **Idempotency keys are first-class.** Every state-changing verb accepts `--idempotency-key` and treats repeats as no-ops returning the prior result (`task run` already does this; extend to `network send` and `me context update`).
- **Exit-code taxonomy:** `0` ok, `1` validation, `2` not-found, `3` permission, `4` conflict (idempotency mismatch), `5` deadline-exceeded, `6` daemon-down, `7` no-data (e.g. `recv --wait --timeout 5s` saw nothing). Today everything is `0` or `1`.

### 4.1 Agent-self surface — NEW namespace `agh me`

| Command | Status | Args / flags | Returns | Why |
|---|---|---|---|---|
| `agh me` | NEW | — | session id, agent name, role, workspace, joined channels, current task run id, peer card | replaces `whoami`'s env-echo with a real round-trip. One command, full context. |
| `agh me context` | NEW | `--include task,inbox,memory,peers` | structured "what should I be working on" payload | Paperclip-style heartbeat. Single fetch instead of three. |
| `agh me capabilities set --from-file caps.json` | NEW | `--diff`, `--digest <sha256>` | updated peer card | lets an agent declare/update its own capability brief at runtime (today only set at session boot) |
| `agh me status --working/--idle/--blocked --detail "..."` | NEW | `--ttl 5m` | none | publishes a `trace` envelope on the agent's current interaction (or no-op if outside one) so peers see live state without crafting JSON. |
| `agh me logout` | NEW | — | none | leaves all channels, ends current task run leases gracefully. Useful for `defer` in agent shells. |

**Implementation hook:** these all resolve via `AGH_SESSION_ID` and a new daemon endpoint `GET /api/sessions/:id/context` that joins session, network manager, task manager, memory.

### 4.2 Channel surface — refactor under `agh ch` (alias of `network`)

Keep `agh network` for operators; add short alias `agh ch` for agents. Add four missing verbs:

| Command | Status | Notes |
|---|---|---|
| `agh ch list` | EXISTS as `agh network channels` | rename short form |
| `agh ch peers [channel]` | EXISTS | unchanged |
| `agh ch join <channel> [--capability cap1,cap2] [--display-name NAME]` | **NEW** | today JoinChannel only fires from session boot via `Manager.JoinChannel` (`internal/network/manager.go:335`); expose a CLI verb that lets a running agent opt into a new channel mid-session. |
| `agh ch leave <channel>` | **NEW** | symmetric. Calls `Manager.LeaveChannel`. |
| `agh ch send --kind say --body '{...}'` | EXISTS as `agh network send` | the `--session` flag becomes optional when `AGH_SESSION_ID` is set; `--channel` defaults to the session's primary channel; add `--idempotency-key` to deduplicate retries. |
| `agh ch recv [--channel C] [--kind say,direct,recipe] [--wait 30s] [--ack]` | **NEW** | the missing primitive. Today only `agh network inbox` returns a snapshot. `recv --wait` long-polls (or streams JSONL) and emits one envelope per line. `--ack` deletes from inbox after read; without it, peek semantics. |
| `agh ch whois <peer>` | **NEW** | publishes a `whois` request and returns the resolved peer card / capability catalog. Today the daemon handles `KindWhois` internally (`internal/network/manager.go:879`) but no CLI verb triggers it. |
| `agh ch discover --capability task.write [--channel C]` | **NEW** | filters peers by capability id from their `NetworkCapabilityBriefPayload`. Pure read; resolves locally from the peer registry — no network round-trip. |
| `agh ch reply --to-message <msg-id> --kind direct --body '{...}'` | **NEW** | sugar that loads the inbound envelope, copies `interaction_id`, `reply_to`, `causation_id`, `trace_id`, `to=from` — eliminates the multi-line guidance string the manager injects today (`internal/network/delivery.go:910-988`). |

### 4.3 Task surface — agent-pool primitives under `agh task`

Existing `agh task` commands stay. Add:

| Command | Status | Notes |
|---|---|---|
| `agh task next [--capability cap] [--channel C] [--wait 30s]` | **NEW** | atomic "claim the next runnable task whose owner is unset and matches my capabilities". Wraps `EnqueueRun → ClaimTaskRun` semantics into one verb. Returns `{task, run, lease_expires_at}`. |
| `agh task pass <run-id> --reason "..." [--to <peer>]` | **NEW** | release the lease without failing. Today `cancel` is overloaded; `pass` clearly means "I don't want this; let someone else try". |
| `agh task lease-extend <run-id> --by 5m` | **NEW** | required for long-running runs to avoid being reclaimed. Today no lease at all. |
| `agh task done <run-id> [--result-file ./out.json]` | **NEW alias** of `task run complete` | shorter, accepts file input so agents don't have to inline-quote JSON. |
| `agh task fail <run-id> --error "..." [--retryable]` | **NEW alias** of `task run fail` | shorter. `--retryable` flag flips a flag the manager can use to requeue. |
| `agh task subscribe <id> [--events status,comment,dependency] [--wait]` | **NEW** | streams JSONL of task lifecycle events. Today only operator-side `observe events` exists. |
| `agh task comment <id> --body "..."` | **NEW** | first-class collaboration verb, mirrors Multica. Stored as a task-scoped event; surfaces in network as `kind: say` with `agh.task_id` ext. |

### 4.4 Sub-agent spawn surface — NEW

| Command | Status | Notes |
|---|---|---|
| `agh spawn --agent <name> [--prompt "..."] [--bind-task <id>] [--bind-channel <c>] [--inherit-memory]` | **NEW** | the missing autonomy primitive. Spawns a child session **owned by the calling session** so the daemon can: (a) auto-cancel children when parent fails; (b) inject parent identity into the child's prompt; (c) auto-claim a task on behalf of the child; (d) optionally enroll the child in the parent's channel(s) with `task.write` capability. Returns `{child_session_id, child_pid, channel, run?}`. |
| `agh spawn --recipe <recipe-id> [--inputs file.json]` | **NEW** | spawns a child wired to execute a published recipe (when AGORA recipe layer lands). |
| `agh agents` | **NEW** | flat list of *live* agents (sessions + their network peer ids), reachable from any agent. Today `agh agent list` shows static defs. Suggest renaming static one to `agh agent definitions`. |

### 4.5 Memory surface — minor polish

`agh memory` is already agent-callable. Two adjustments:

| Command | Status | Notes |
|---|---|---|
| `agh memory write <name>` | EXISTS | when called from inside a session, default `--scope` should be `workspace` and `--workspace` should auto-resolve. |
| `agh memory share <name> --with-channel <c>` | **NEW** | optional: publish a memory file as a `recipe` envelope so other agents on a channel can install it. Bridges memory and network. |

### 4.6 JSON contract notes

- Every `recv`, `next`, `claim`, `subscribe` returns `{ "v": 1, "type": "...", "data": {...} }` so streaming consumers can switch on `type`.
- Errors return `{ "error": { "code": "validation", "message": "...", "details": {...} } }` — agents can `jq -e` on `.error.code`.
- Add `Idempotency-Key` HTTP header pass-through on the UDS layer for `network send` and `me context update`.

---

## 5. Implementation hooks

| Need | Touch | Notes |
|---|---|---|
| Implicit identity resolution | `internal/cli/root.go` (extend `commandDeps` with `defaultSessionFromEnv`), every command's flag parsing | one helper `resolveIdentity(cmd, deps) (sessionID, channel, agentName, error)` consumed by `network send`, `network inbox`, `task next`, `task done`, `me`, `spawn`. |
| `agh me context` | new `internal/api/contract` type `MeContextPayload`; new handler in `internal/api/udsapi/sessions.go` (`GET /api/sessions/:id/context`); aggregates from `session.Manager`, `network.Manager`, `task.Manager`, `memory.Catalog`. | also surfaced on HTTP for the web UI's "agent inspector". |
| `agh ch join/leave` | `internal/api/udsapi/network.go` add `POST /api/network/channels/:channel/join` & `…/leave`; route into `Manager.JoinChannel/LeaveChannel`. | `JoinChannel` already exists (`internal/network/manager.go:335`); just expose. |
| `agh ch recv --wait` | needs a long-poll/SSE endpoint `GET /api/network/inbox/stream?session=...&kind=...`; `internal/network/delivery.go` already has a per-session worker, surface a fan-out channel on it. | `--ack` writes back via `POST /api/network/inbox/:envelope_id/ack`. Without it, inbox stays as today (peek). |
| `agh ch whois`, `agh ch discover` | `whois` adds `POST /api/network/whois` (emits envelope, blocks on reply); `discover` is a pure read of `peers.RemotePeers` filtered by capability — already partially there in `network.PeerInfo`. | Capabilities live in `internal/network/capability_brief.go`. |
| `agh ch reply` sugar | pure CLI-side: load envelope from inbox, derive flags, call existing `NetworkSend`. | Removes the need for the multi-line guidance string in `internal/network/delivery.go:820-988`. |
| `agh task next` | new `internal/task/manager.go` method `NextRun(ctx, NextRunQuery{Capability, Channel, OwnerHint})`; uses existing `Enqueue → Claim` plumbing inside one transaction. | Enforce: caller must declare the capability in its peer card. |
| `agh task pass` and `agh task lease-extend` | new `task.Run` state transitions; schema column `lease_expires_at` already implied by claim. Verify in `internal/store/schema.go`. | Add to `internal/api/udsapi/routes.go` under `task-runs`. |
| `agh task subscribe` | use existing `StreamTask` SSE handler (`internal/api/udsapi/routes.go: tasks.GET("/:id/stream", ...)`) — wire CLI to it. | Just a CLI wrapper. |
| `agh spawn` | new command + new daemon endpoint `POST /api/sessions/:parent_id/children`. Calls `session.Manager.CreateSession` with `parent_session_id` + optional `bound_task_id` + `bound_channel`. | Touches `internal/session/manager.go`, `internal/api/contract/contract.go` (`CreateChildSessionRequest`). Lifecycle: parent-cascade on stop. |
| `agh agents` (live) | small read on `session.Manager.Sessions()` joined with `network.Manager.LocalPeers()`. | Naming: rename existing `agh agent` → `agh agent definitions` to free `agh agents`. |
| Identity-aware audit | `internal/api/udsapi/server.go` middleware: tag every incoming UDS request with the calling session id (from a new `X-Agh-Session` header set by the CLI when env vars are present). | Lets `internal/network/audit.go` attribute writes to specific agents instead of "the daemon did it". |
| Idempotency for `network send` | `internal/network/manager.go: Send` accept optional `IdempotencyKey`; dedup window 5 min in `runtimeStats`. | Mirrors what `EnqueueRun` already does. |
| Exit-code taxonomy | `internal/cli/root.go: ExecuteContext`; map daemon error codes → exit codes via a small switch. | One file change; agents finally get scriptable failure modes. |
| `--output jsonl` | `internal/cli/format.go` — add `OutputJSONL`. Emit one `outputBundle.jsonValue` per item for streaming endpoints. | Required for `recv --wait`, `task subscribe`, `task next --wait`, `me context --watch`. |

CI-enforceable boundary: keep all of this inside `internal/cli` and `internal/api/udsapi`. No package outside `daemon/` should learn about `AGH_*` env vars.

---

## 6. Open questions

1. **Auth model.** Today the UDS itself is the security boundary — anyone with the socket is trusted. When agents are spawned by the daemon, the daemon already knows their PID and session id; should the CLI carry a per-session token (written to a file the spawn injects via env)? Without it, malicious tools running inside an agent's sandbox could impersonate any session. For Phase 3 (network), this becomes a hard requirement — proposal: per-session HMAC key in `~/.agh/sessions/<id>/token`, exported as `AGH_SESSION_TOKEN`, validated by middleware.

2. **Sync vs async.** `agh ch recv --wait` and `agh task next --wait` need a long-poll or SSE. Options:
   - SSE over UDS (already used for `session/stream`, `task/stream`). Simple, works.
   - Returning JSONL on a single HTTP response with chunked encoding. Slightly easier to consume from `bash | while read line`.
   - A `agh inbox` daemon-mode subprocess that prints to stdout and stays alive (Multica-style). Heavier but most ergonomic for shell agents. Recommend: SSE first, JSONL transport as `--output jsonl` flag, daemon-mode later.

3. **Output stability contract.** Agents will start to depend on JSON shapes. We need a policy: **`-o json` is a stable contract; `-o human` and `-o toon` are not.** Document this in `CLAUDE.md` and add `-o jsonl` for streaming. JSON Schemas for every CLI response in `internal/api/contract/` should be promoted to first-class generated artifacts (the OpenAPI in `openapi/agh.json` already covers HTTP — extend coverage).

4. **Capability declaration.** Today capabilities are baked at session boot via `NetworkPeerCapability`. Should agents be able to *acquire* capabilities at runtime (e.g., "I just installed a skill that adds `pdf.parse`")? Proposal: `agh me capabilities add` mutates the peer card and re-publishes a `greet` so peers see the change.

5. **Permission scoping.** When agent A spawns child agent B with `agh spawn`, should B inherit A's capability set? Default to **no** (least privilege); allow `--inherit-capabilities` flag. Document in spawn handler.

6. **`agh me logout` semantics.** Does it terminate the session or just leave channels? Proposal: leave channels + end task lease only; session continues until ACP exits. Avoids killing the agent's own process via its own shell.

7. **Recipe verb.** AGORA spec v0.2 adds a `recipe` kind. Proposal places `agh ch send --kind recipe --body @recipe.json` and `agh ch recv --kind recipe --wait` as the natural plumbing. No new top-level verb needed — recipes are envelopes. Confirm with Phase 3 RFC owner before implementing `agh recipe install` shortcuts.

8. **Process supervision of spawned agents.** If `agh spawn` returns a child session id, how does the parent learn the child failed? Options: parent gets a `kind: trace` envelope on a private channel, or `agh task subscribe` covers it. Recommend the latter — keep network-layer events on the network, keep task lifecycle on tasks.

---

### Closing observation

The existing daemon already implements 80 % of the machinery an autonomous agent needs (channels, peers, capability briefs, task-run state machine, network manager with audit). The missing 20 % is **CLI ergonomics for agents**: implicit identity, blocking primitives (`recv --wait`, `task next --wait`), reply sugar that consumes the guidance the daemon already paints into envelopes, a `me` namespace, and a `spawn` verb. Every gap above maps to a small surface change with no architectural disruption.
