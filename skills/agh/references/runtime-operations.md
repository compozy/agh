# Runtime Operations

## Contents

- Operating model
- Session lifecycle
- Session CLI
- Diagnostics order
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

## Session CLI

Use structured output when agents need to inspect or route results.

    agh session new --agent general --name review-run
    agh session new --agent codex --cwd /absolute/path/to/worktree --name fix-task
    agh session list --all -o json
    agh session status <session-id> -o json
    agh session events <session-id> --follow
    agh session history <session-id>
    agh session prompt <session-id> "Summarize the last three tool results."
    agh session stop <session-id>
    agh session resume <session-id>
    agh session wait <session-id>

If an AGH-native session tool is visible, prefer the tool because it is policy-aware and easier for the daemon to audit. Use the CLI when the tool is denied, absent, or explicitly requested.

## Diagnostics Order

When a session behaves unexpectedly:

1. Run agh session status <id> -o json to classify lifecycle and provider state.
2. Read agh session events <id> for startup, prompt, tool, stop, and error events.
3. Read agh session history <id> for turn-grouped output.
4. Check workspace and agent resolution if the wrong prompt, tools, or skills appear.
5. Check provider command availability only after daemon-side state is understood.

Do not treat stale UI state, chat messages, or memory notes as runtime authority.

## Runtime Boundaries

AGH must remain agent-manageable. Any runtime capability that affects state should have a deterministic CLI, HTTP/UDS, or tool path with machine-readable output. UI-only management is incomplete.

Management flows involving daemon lifecycle, raw secrets, OAuth, trust roots, provider bootstrap, destructive repair, and cross-session terminal-state mutation stay on operator surfaces unless AGH explicitly exposes a scoped tool for them.
