# Runtime Operations

## Contents

- Operating model
- Session lifecycle
- Session CLI
- Diagnostics order
- Status, doctor, logs, and support
- Runtime boundaries

## Operating Model

AGH is a local-first daemon that starts ACP-compatible agents as managed subprocesses, records events, and exposes runtime control through CLI, HTTP/SSE, UDS, and agent tools. Treat the daemon as the source of truth for sessions, events, task state, network rooms, memory, skills, and extension resources.

Do not manage runtime state by editing SQLite databases, direct NATS subjects, process internals, or generated projections. Use public AGH surfaces with structured output.

## Session Lifecycle

AGH sessions are daemon-owned runtimes. Common states:

- starting - the daemon accepted the session and is booting the provider.
- active - the provider is connected and ready for prompts.
- stopping - shutdown has started.
- stopped - the runtime exited and can be inspected or resumed.

Session types include user sessions and daemon-managed sessions such as dream, system, coordinator, worker, and reviewer sessions. Do not infer authority from a session type alone. Use the session context and daemon tools to confirm what the current session may do.

Attachability is explicit runtime state. Use `agh session list --resumable -o json` and `agh session resume` instead of assuming a stopped or idle session can be reused.

After prompt admission, the daemon owns the turn lifetime. Closing a browser tab, navigating away from the web app, dropping an SSE stream, or disconnecting a CLI/UDS response only detaches that viewer; it does not cancel the accepted prompt. Use explicit runtime intent such as `agh session stop`, prompt cancel, or interrupt controls when cancellation is required.

The event store and transcript are the durable source of truth for reattach. When reconnecting to an existing session, read `agh session history <session-id>` or the transcript API first, then follow session events from the latest cursor. Do not reconstruct session state from UI cache, memory notes, or JSONL sidecars.

## Session CLI

Use structured output when agents need to inspect or route results.

    agh session new --agent general --name review-run
    agh session new --agent codex --cwd /absolute/path/to/worktree --name fix-task
    agh session list --all -o json
    agh session list --resumable -o json
    agh session status <session-id> -o json
    agh session health <session-id> -o json
    agh session inspect <session-id> --include-wake-events -o json
    agh session recap <session-id> --limit 20 -o json
    agh session events <session-id> --follow
    agh session history <session-id>
    agh session prompt <session-id> "Summarize the last three tool results."
    agh session stop <session-id>
    agh session resume <session-id>
    agh session resume --latest --workspace checkout-api
    agh session repair <session-id> --dry-run -o json
    agh session soul refresh <session-id> --expected-digest sha256:old -o json
    agh session approve <session-id> --request-id req_123 --turn-id turn_123 --decision allow-once
    agh session wait <session-id>

If an AGH-native session tool is visible, prefer the tool because it is policy-aware and easier for the daemon to audit. Use the CLI when the tool is denied, absent, or explicitly requested.

## Onboarding State

First-run onboarding completion is a global instance flag (stored in the `app_metadata` table, not per-workspace). Inspect or manage it through the CLI or the HTTP/UDS `/api/onboarding` endpoints:

    agh onboarding status -o json
    agh onboarding complete    # mark first-run onboarding as done
    agh onboarding reset       # clear the flag so the web wizard runs again

The web first-run wizard blocks the dashboard until this flag is set. Resetting it surfaces the wizard again on next load. Fresh daemon boot registers the operator `$HOME` as the default workspace before the wizard starts, so the workspace step should not require manual project registration on a clean machine.

Native session tools are read-oriented. Recap, repair, approval, session inspect, and Soul refresh are CLI/HTTP management flows unless the live registry exposes a scoped native tool.

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

`agh status -o json` is the consolidated runtime status surface for daemon health, providers, MCP servers, config apply status, and log tail summary. `agh doctor -o json` runs diagnostic probes; `--only`, `--exclude`, and `--quiet` bound the probe set for agents.

`agh logs --follow -o jsonl` streams redacted runtime logs over SSE. Use filters such as `--session`, `--workspace`, `--run`, `--actor kind:id`, `--provider`, `--component`, `--outcome`, and `--error-only` before broad log reads.

`agh support bundle --yes` creates and downloads a redacted support bundle. It may include status, doctor, provider, event-summary, log-tail, and config-apply snapshots unless `--no-status` is passed. Treat support bundles as operator artifacts, not native tool calls.

## Runtime Boundaries

AGH must remain agent-manageable. Any runtime capability that affects state should have a deterministic CLI, HTTP/UDS, or tool path with machine-readable output. UI-only management is incomplete.

Management flows involving daemon lifecycle, raw secrets, OAuth, trust roots, provider bootstrap, destructive repair, and cross-session terminal-state mutation stay on operator surfaces unless AGH explicitly exposes a scoped tool for them.
