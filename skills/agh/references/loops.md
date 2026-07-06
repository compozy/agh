# Loops

Agent operation guidance for AGH Loops — the deterministic goal → verify → stop programs the daemon
owns and runs. Use this reference when you author, configure, run, observe, approve, or stop a Loop
from inside AGH. Prefer the native `agh__loop_*` tools; fall back to `agh loop` CLI or HTTP with
structured output. Never guess a schema — resolve `agh__tool_info` for the exact descriptor first.

## Contents

- The tool set and CLI verbs
- The authoring loop
- Terminal outcomes and live states
- The approve capability gate
- Reference grammar and reserved action kinds
- Loop hook events
- Watch-source behavior
- Harvesting a channel decision

## The Tool Set And CLI Verbs

Toolset `agh__loops` — 13 native tools. Every tool has a matching `agh loop` verb; the CLI adds one
verb (`edit`) that has no native tool.

| Native tool           | Mode                            | CLI                  | Purpose                                                         |
| --------------------- | ------------------------------- | -------------------- | --------------------------------------------------------------- |
| `agh__loop_list`      | read                            | `agh loop list`      | List Loop definitions in the workspace.                         |
| `agh__loop_inspect`   | read                            | `agh loop inspect`   | Read one definition: inputs, contract, start bindings, version. |
| `agh__loop_validate`  | read                            | `agh loop validate`  | Lint + compile a definition without saving.                     |
| `agh__loop_status`    | read                            | `agh loop status`    | Read one run's status with generation detail.                   |
| `agh__loop_runs`      | read                            | `agh loop runs`      | List runs in the workspace.                                     |
| `agh__loop_create`    | mutating                        | `agh loop create`    | Create/fork, or CAS-publish when `expected_version` is set.     |
| `agh__loop_run`       | mutating                        | `agh loop run`       | Start a run, or dry-run with `dry: true` / `--dry-run`.         |
| `agh__loop_configure` | mutating                        | `agh loop configure` | Write per-Loop runtime config overrides.                        |
| `agh__loop_pause`     | mutating                        | `agh loop pause`     | Request a generation-boundary pause.                            |
| `agh__loop_resume`    | mutating                        | `agh loop resume`    | Resume a paused or pause-requested run.                         |
| `agh__loop_approve`   | mutating · **capability-gated** | `agh loop approve`   | Apply one human-gate decision.                                  |
| `agh__loop_stop`      | destructive                     | `agh loop stop`      | Stop one active run.                                            |
| `agh__loop_delete`    | destructive                     | `agh loop delete`    | Delete a writable workspace definition.                         |

There is **no `agh__loop_edit` native tool**. Agents edit a definition through the authoring loop
(validate → dry-run → `agh__loop_create` with `expected_version`) or by a filesystem write. The CLI
`agh loop edit` is a `$EDITOR` convenience for operators and publishes through the same
compare-and-swap path.

## The Authoring Loop

Follow **inspect → validate → dry-run → publish (with `expected_version`) → run**. Every step before
`run` is structured and spends no tokens.

1. **inspect** — `agh__loop_inspect` returns the definition and its current `version`. Read the
   version before you change anything.
2. **validate** — `agh__loop_validate` lints and compiles a candidate without saving; it returns
   per-node codes (`unknown_reference`, `node_id_invalid`, `verdict_policy_requires_judge`,
   `fan_out_ceiling_exceeded`).
3. **dry-run** — `agh__loop_run` with `dry: true` resolves inputs and returns the first generation's
   plan without creating a run or spending budget.
4. **publish** — `agh__loop_create` with `expected_version` set to the version from step one (or
   HTTP `PATCH /loops/:name`). This is compare-and-swap: a stale version is rejected `409` with the
   current version. Use PATCH/create-with-version for **all** programmatic editing — the filesystem
   write path is last-write-wins and unsafe for concurrent agents.
5. **run** — `agh__loop_run`. Only now does the Loop spend tokens.

New Loops start as a fork (`agh__loop_create` with `fork_from_name`); there is no blank-canvas
authoring. Read-only sources — including the default `dev-cycle` Loops — must be forked before you
adapt them.

## Terminal Outcomes And Live States

A run holds one of eleven states. Report the terminal outcome exactly — never round an error or an
exhausted budget up to success.

**Terminal (6):**

- `done` — the goal was verified. The only success outcome.
- `no_op` — ran, found nothing to do. A clean watch tick is `no_op`, not a fake `done`.
- `blocked` — an external dependency blocked progress (missing dependency/credential/resource, a
  human-gate `reject`, or a `loop.gate.pre` denial).
- `failed` — an unrecoverable node/gate error, a `loop.generation.pre` denial, or an operator
  `Stop` (truthful cause `operator_stop`).
- `exhausted` — the iteration cap or fan-out ceiling tripped before the goal.
- `stalled` — no progress: the no-progress window elapsed, the failure circuit breaker tripped, the
  blocker-ID signature repeated, or a watched source went silent.

**Live (5):** `queued` (deferred start under `concurrency: queue`), `running`, `watching` (dormant
watch tick), `needs-approval` (parked on a human gate — a live pause, not terminal), `paused`
(operator paused at a boundary). `ready` and `awaiting_child` are node-level, never run states.

## The Approve Capability Gate

`agh__loop_approve` requires the `loops.approve` capability, and **an agent can never approve a run
it started**. The daemon compares the approver's identity against the run's starter: an agent
session cannot approve its own run — the call is denied `ErrPermissionDenied` (reason
`approval_self_denied`). A different agent, or an operator, can approve. Provide `run_id`, `gate_id`,
and `decision` (`approve` | `request_changes` | `reject`). `approve` resumes, `request_changes`
revises into the next generation, `reject` halts on a `blocked` outcome.

## Reference Grammar And Reserved Action Kinds

Definitions reference data over one namespace with two surfaces, chosen by the field:

- **Values** — Go `{{ }}` templates in string fields (`params.*`, rubrics, `transform.map.*`).
- **Conditions** — CEL returning `bool` (`branch.condition`, `fan-out.filter`, `contract.stop_when`).

Namespace roots: `inputs.<name>`, `nodes.<id>.output.<path>`, `nodes.<id>.status`, `item`/`index`
(fan-out scope only), `trigger.<path>` (trigger/webhook starts only), `generation`. Node IDs match
`^[a-z][a-z0-9_]*$` (lowercase snake_case) so the same ID is valid in both surfaces.

Node classes: `action` (open), `control` (closed), `source` (closed). Reserved **action** kinds are
`run-agent`, `run-loop`, `transform`; every other action kind is a literal tool ID
(`agh__*`/`ext__*`/`mcp__*`). Control kinds: `fan-out`, `collect`, `branch`, `gate`, `sub-loop`.
Source kinds: `input`, `file-import`, `watch-source`. A gate's `verdict_policy: revise_until_clean`
requires an `agent-judge` or `human` criterion.

## Loop Hook Events

The `loop.*` hook family has seven events; two can block. Dispatch is typed and fail-open — a broken
hook does not fail a run.

- `loop.started`, `loop.generation.post`, `loop.gate.post`, `loop.node.terminal`, `loop.terminal` —
  observe-only.
- `loop.generation.pre` — sync-eligible; a denial ends the run `failed`.
- `loop.gate.pre` — sync-eligible; a denial ends the run `blocked`.

Every payload carries the loop context (`loop_run_id`, `workspace_id`, `loop_name`, `generation`,
`node_id`, and more). Manage them with `agh__hooks_*`.

## Watch-Source Behavior

A Loop with a `watch-source` node is a watch Loop. It holds `watching` between ticks, defaults to
`iteration_cap: 0` (`∞`, never `exhausted`), ends a clean tick `no_op`, and ends on silence past its
window `stalled` (reason `watch_source_silence`). The default `dev-cycle` `reviews-watch` Loop is a
watch Loop and requires `gh` to be installed and authenticated for CodeRabbit polling.

## Harvesting A Channel Decision

To let agents converse and act on the result, post with an `agh__network_send` action carrying a
`harvest: { kind: channel_result, window, responder?, content_rule? }`. The retired `channel-post`
kind does not exist. After the send, the node waits `window` for the designated result — a `say`
with `intent: result` or a `trace` with `state: completed` — and exposes it as
`nodes.<id>.output.*`. Silence past `window` ends the run `stalled`. `content_rule` narrows the
match: `any`, `json`, `non_empty`, `contains:<needle>`, or `json_path:<a.b.c>`. This capability is a
documented example, not a packaged default Loop.
