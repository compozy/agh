# Runtime Operations

## Operating Model

AGH is a local-first daemon that starts ACP-compatible agents as managed subprocesses, records events, and exposes runtime control through CLI, HTTP/SSE, UDS, and agent tools. Treat the daemon as the source of truth for sessions, events, task state, network rooms, memory, skills, and extension resources.

Do not manage runtime state by editing SQLite databases, process internals, or generated projections. Use public AGH surfaces with structured output.

## Session Lifecycle

AGH sessions are daemon-owned runtimes. Common states:

- starting - the daemon accepted the session and is booting the provider.
- active - the provider is connected and ready for prompts.
- stopping - shutdown has started.
- stopped - the runtime exited and can be inspected or resumed.

Session types include user sessions and daemon-managed sessions such as dream, system, coordinator, worker, and reviewer sessions. Do not infer authority from a session type alone. Use the session context and daemon tools to confirm what the current session may do.

An unnamed user session receives a daemon-owned durable title from its first submitted prompt. An explicit session name is preserved, and daemon-managed session types are not retitled by ordinary prompt submission. Treat the persisted session name as catalog identity; do not synthesize a competing title from client cache or transcript state.

Attachability is explicit runtime state. Use `agh session list --resumable -o json` and `agh session resume` instead of assuming a stopped or idle session can be reused.

After prompt admission, the daemon owns the turn lifetime. Closing a browser tab, navigating away from the web app, dropping an SSE stream, or disconnecting a CLI/UDS response only detaches that viewer; it does not cancel the accepted prompt. Use explicit runtime intent such as `agh session stop`, prompt cancel, or interrupt controls when cancellation is required.

The event store and materialized transcript are the durable source of truth for reattach. Transcript GET returns the newest bounded `entries` page plus `epoch`, `generation`, `max_sequence`, and `has_older`; request older entries with `before_sequence=next_before_sequence`. Each entry carries immutable `start_sequence` identity and its latest shaping `sequence`. Do not reconstruct session state from UI cache, memory notes, or JSONL sidecars.

The HTTP/UDS stream defaults to `transcript_snapshot`, batched `transcript_delta`, and terminal `session_stopped` frames. Reconnect with the last SSE cursor plus the snapshot's `epoch` and `generation`; a fence mismatch returns an explicit reset snapshot. The removed `replay` query is invalid. Use `frames=raw` for persisted `SessionEventPayload` rows; `agh session events --follow` already requests raw frames.

## Session CLI

Use structured output when agents need to inspect or route results.

    agh session new --agent general --name review-run
    agh session new --agent codex --cwd /absolute/path/to/worktree --name fix-task
    agh session list --all -o json
    agh session list --type user --state active --sort last_activity -o json
    agh session list --resumable -o json
    agh session status <session-id> -o json
    agh session health <session-id> -o json
    agh session inspect <session-id> --include-wake-events -o json
    agh session recap <session-id> --limit 20 -o json
    agh session events <session-id> --follow
    agh session events <session-id> --after 42
    agh session history <session-id>
    agh session history <session-id> --last 20 --after 42
    agh session prompt <session-id> "Summarize the last three tool results."
    agh session stop <session-id>
    agh session remove <session-id>
    agh session resume <session-id>
    agh session resume --latest --workspace checkout-api
    agh session repair <session-id> --dry-run -o json
    agh session soul refresh <session-id> --expected-digest sha256:old -o json
    agh session approve <session-id> --request-id req_123 --turn-id turn_123 --decision allow-once
    agh session wait <session-id>

If an AGH-native session tool is visible, prefer the tool because it is policy-aware and easier for the daemon to audit. Use the CLI when the tool is denied, absent, or explicitly requested.

`agh session stop` preserves durable history and resume state. `agh session remove` is destructive: it stops an active runtime when necessary, then removes the catalog row and persisted session directory. Use removal only when the operator intends to discard that history.

The session catalog is counted and workspace-scoped. Dream sessions are internal and never appear in catalog results. HTTP and UDS clients can filter exact public session type with `type=user|system|coordinator|spawned`; the CLI exposes the same filter as `--type`. Browser integrations should subscribe once to `/api/sessions/catalog-stream`, route each wake signal by its authoritative `workspace_id`, and refetch that workspace's catalog page instead of incrementing local counters.

## Onboarding State

First-run onboarding completion is a global instance flag (stored in the `app_metadata` table, not per-workspace). Inspect or manage it through the CLI or the HTTP/UDS `/api/onboarding` endpoints:

    agh onboarding status -o json
    agh onboarding complete    # mark first-run onboarding as done
    agh onboarding reset       # clear the flag so the web wizard runs again

The web first-run wizard blocks the dashboard until this flag is set. Resetting it surfaces the wizard again on next load. Fresh daemon boot registers the operator `$HOME` as the default workspace before the wizard starts, so the workspace step should not require manual project registration on a clean machine.

Native session tools are read-oriented. Recap, repair, approval, session inspect, and Soul refresh are CLI/HTTP management flows unless the live registry exposes a scoped native tool.

## Messaging Bridge Delivery and Progress

Bridge instances own their delivery behavior. Manage them through `agh bridge`, `/api/bridges`, or the equivalent UDS endpoints; do not edit extension state or storage directly. Tool progress is presentation-only bridge delivery data and never becomes session transcript or ACP history.

Use `agh bridge manifest slack --instance <id>` for Slack app setup and `agh bridge setup whatsapp|telegram|discord` for guided, write-only secret binding. Teams, Google Chat, GitHub, and Linear have no setup subcommand: use `agh bridge create --enabled=false`, bind exact slots with `agh bridge secret-bindings put`, verify, enable, then verify public reachability. Generic binding accepts secret contents through `--secret-value-stdin` or `--secret-value-file <path>` and rejects inline secret arguments. The setup commands accept strict headless JSON through the global `--json` flag; supplied and existing secrets are never echoed. WhatsApp verify tokens and Telegram `--print-only` webhook secrets must be supplied or generated with the explicit one-run `--reveal-generated-secrets` JSON disclosure, so an operator-needed value never becomes irretrievable. Telegram setup otherwise registers `setWebhook` through the daemon.

An empty bridge `dm_policy` normalizes to permissive `open`. The current create/update CLI has no DM-policy flag; keep the bridge disabled and use the Web editor or `PATCH /api/bridges/:id` to set `allowlist` or `pairing` plus the complete `provider_config.dm` lists. `pairing` consumes pre-populated `paired_user_ids`/`paired_usernames` with allowlist fallback; it does not enroll or approve senders interactively. DM policy does not govern groups, channels, spaces, repositories, or issues.

`agh bridge verify <id> --json` asks the owning adapter for typed `pass|warn|fail|skipped` checks without changing instance lifecycle state; GitHub and Linear currently skip identity, so enabled runtime health owns their live auth result. Any failed check makes the command nonzero after it writes the records. `agh doctor --only bridge --json` aggregates the same checks. After enablement, `agh bridge send-test <id> --message ...` makes a real provider delivery. `agh bridge test-delivery` remains a target-resolution dry run and sends nothing. HTTP and UDS expose `GET /api/bridges/providers/slack/manifest?instance=<id>` plus `POST /api/bridges/:id/verify`, `/send-test`, and `/webhook/register`; there is no `/api/bridges/setup` route.

Credential-bearing API, OAuth, and service destinations are operator-owned adapter environment, not instance configuration. `provider_config` rejects `api_base_url`, `oauth_token_url`, `service_url`, `openid_metadata_url`, and `token_url`; use the provider's `AGH_BRIDGE_*` process variables for trusted overrides. Provider clients use `bridgesdk.CredentialedHTTPClient`, returning the original `3xx` for classification without forwarding credentials or replaying mutation bodies. `webhook.public_url` must be public HTTPS, and verification blocks internal/special-use addresses, proxying, and redirects before reachability is attempted. Bridge reads expose the validated callback as optional `webhook_public_url`; clients use that projection for setup readiness instead of re-parsing `provider_config`.

Terminal replies are split provider-side on natural boundaries with `(N/M)` markers. AGH measures Slack at 40,000 UTF-16 code units, Telegram at 4,096 UTF-16 code units, Discord at 2,000 Unicode code points, Teams at 28,000 Unicode code points, Google Chat at 32,000 UTF-8 bytes, and WhatsApp at 4,096 Unicode code points. Every multi-chunk delivery acknowledges its last remote message. Edit-capable providers keep an oversized non-terminal response in one mutable preview and materialize its continuations only on the terminal update. Slack converts common Markdown to mrkdwn; Telegram sends escaped MarkdownV2 and retries a typed parse rejection as plain text.

Configure the typed `delivery_defaults.progress` block with `tool_progress` (`off`, `new`, `all`, or `verbose`), `grouping` (`accumulate` or `separate`), `typing`, and `reactions`. Slack, Telegram, and Discord default to `new` plus `accumulate` with typing and reactions enabled; other platforms default to `off` plus `accumulate` with both affordances disabled unless the instance overrides them. `new` deduplicates consecutive starts but still emits completed and failed phases.

Slack, Telegram, Discord, Teams, and Google Chat can update an accumulated progress bubble; Slack and Telegram apply their platform dialects to the daemon-rendered line. WhatsApp is append-only, so prefer `new` plus `separate` when enabling its sparse one-line statuses. GitHub and Linear acknowledge progress without writing to issues.

The CLI exposes the same fields on `bridge create` and `bridge update`:

    --delivery-progress <off|new|all|verbose>
    --delivery-progress-grouping <accumulate|separate>
    --delivery-progress-typing[=true|false]
    --delivery-progress-reactions[=true|false]

Use structured output to inspect the saved resource after mutation. An adapter that has not registered a progress handler acknowledges these events without a provider-side effect; final answer delivery remains independent. Progress previews are daemon-rendered and redacted before they cross the extension boundary.

Supported Slack and Telegram message edits reach the agent as a typed `edit` prompt block. Slack, Telegram, and Google Chat replies include quoted parent text and author only when an embedded snapshot or the bounded workspace/instance/conversation cache has it; a miss stays empty and never triggers a provider fetch. At startup, AGH reconciles durable in-flight delivery checkpoints before accepting new prompt or registration side effects. The ledger contains routing, sent/acknowledged sequence, remote-message, terminal, and aggregate-metric state—not streamed response or progress text. A sequence sent but not acknowledged is terminalized locally as indeterminate with no provider replay. An unfinished row without an unmatched send intent gets one write-ahead terminal error post, including on append-only providers or when its old remote anchor no longer exists.

A bridge route reuses its active AGH session. If that session is busy, ingress retries admission locally up to three times within roughly five seconds; it does not automatically queue, interrupt, or steer the turn. Stop the route's `session_id` with `agh session stop` when the next accepted provider event must start a clean session; the old transcript remains history and the route rebinds to a replacement.

Distinguish restart recovery from a remotely committed mutation whose required result was unavailable. After a provider accepts a mutation but its response or required ID cannot be materialized, adapters return `CommittedMutationError`; bridgesdk emits `committed_result_unavailable`, and the broker records a terminal error without replay or a fabricated remote ID. `agh bridge send-test --json` reports that status, omits `remote_message_id`, and returns a redacted error; it is a direct control probe, so branch on `status` rather than command success and do not expect a broker ledger row or health-failure metric. Inspect the provider conversation before any manual resend because the remote artifact may already exist. An indeterminate progress mutation drops only that progress bubble; later final text remains eligible.

## Diagnostics Order

When a session behaves unexpectedly:

1. Run `agh session status <id> -o json` to classify lifecycle and provider state.
2. Run `agh session health <id> -o json` or `agh session inspect <id> --include-wake-events -o json` when wake policy, stale health, or Heartbeat state is relevant.
3. Read `agh session events <id>` for startup, prompt, tool, stop, and error events.
4. Read `agh session history <id>` or `agh session recap <id> -o json` for turn-grouped output and deterministic recent context.
5. Check workspace and agent resolution if the wrong prompt, tools, or skills appear.
6. Run `agh doctor -o json` and only then check provider command availability or external auth state.
7. Use `agh session repair <id> --dry-run -o json` before any repair write.

Do not treat stale UI state, chat messages, or memory notes as runtime authority.

## Status, Doctor, Logs, And Support

`agh status -o json` is the consolidated runtime status surface for daemon health, providers, MCP servers, config apply status, schema migration streams, and log tail summary. Inspect `schema_streams` after startup to confirm that the global and memory streams report their expected version, applied migration count, and digest. An incompatible daemon-global `agh.db` is refused during boot, before readiness. By contrast, an incompatible per-session `events.db` can be discovered after the daemon is ready when a reader such as `agh session history <id> -o json` or `GET /api/workspaces/{workspace_id}/sessions/{session_id}/history` opens that session; that operation fails without making the healthy daemon-global store unavailable.

For `legacy_database`, stop AGH, cold-move the complete containing `AGH_HOME` or workspace `.agh` family, and select a separate fresh home. Preserve every sibling database and SQLite sidecar together; never edit migration history or move one live file. For `schema_ahead`, first use a newer compatible AGH binary against the stopped, intact family—the state-preserving recovery. Use a fresh home only if discarding that state is acceptable. Stopped-daemon provider-auth, extension, and MCP-auth direct opens emit one JSON error document with `diagnostic.code` set to `legacy_database` or `schema_ahead`; use its surface and canonical-path evidence instead of parsing prose. `agh doctor -o json` runs diagnostic probes; `--only`, `--exclude`, and `--quiet` bound the probe set for agents.

`agh logs --follow -o jsonl` streams redacted runtime logs over SSE. Use filters such as `--session`, `--workspace`, `--run`, `--actor kind:id`, `--provider`, `--component`, `--outcome`, and `--error-only` before broad log reads.

`agh support bundle --yes` creates and downloads a redacted support bundle. It may include status, doctor, provider, event-summary, log-tail, and config-apply snapshots unless `--no-status` is passed. Treat support bundles as operator artifacts, not native tool calls.

## Runtime Boundaries

AGH must remain agent-manageable. Any runtime capability that affects state should have a deterministic CLI, HTTP/UDS, or tool path with machine-readable output. UI-only management is incomplete.

Management flows involving daemon lifecycle, raw secrets, OAuth, trust roots, provider bootstrap, destructive repair, and cross-session terminal-state mutation stay on operator surfaces unless AGH explicitly exposes a scoped tool for them.

Marketplace catalog configuration is global-only because its projection and refresh service are global. `agh__config_set` and `agh__config_unset` may change `marketplace.catalog.ttl` and `marketplace.catalog.timeout` at global scope; each mutation runs the daemon settings apply lifecycle and returns the real `applied`, `apply_record_id`, `active_generation`, `next_action`, and reconciliation diagnostics. `marketplace.catalog.base_url` is a trust root and remains operator-only through global `agh config set`. Workspace overlays and workspace-scoped writes are rejected.
