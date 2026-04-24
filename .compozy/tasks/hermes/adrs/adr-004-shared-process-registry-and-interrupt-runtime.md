# ADR-004: Own Tool Processes and Interrupts in a Shared Runtime Package

## Status

Accepted

## Date

2026-04-24

## Context

AGH currently manages terminal subprocesses inside ACP/local tool host internals, hooks in hook-specific executors, and runtime subprocesses in separate packages. Selected Hermes issues require a process registry with checkpoint-on-write semantics, PID/start-time verification on boot, and per-thread or per-turn tool interrupts.

Placing this behavior only in `session.Manager` would miss hooks, extensions, and non-session-owned processes. Placing it inside `environment.ToolHost` would mix file/permission/terminal APIs with durable runtime ownership.

## Decision

Create a shared runtime package for process registry and interrupts, such as `internal/toolruntime` or `internal/runtime/processes`.

The package will own:

- `ProcessRegistry` with checkpoint-on-write persistence.
- PID and start-time validation using `procutil`.
- Boot reconciliation for stale, live, and orphaned process records.
- Process ownership metadata: session ID, turn ID, tool call ID, terminal ID, source, command, cwd, start time, and state.
- `InterruptController` scoped to session, turn/thread, and tool process ownership.

ACP terminal management, local and remote environment tool hosts, hooks, extension host APIs, and future tool runtimes will integrate with this shared package.

## Alternatives Considered

- Extend `environment.ToolHost` as the owner. This avoids a new package but overloads the interface and leaves hooks/extensions awkward.
- Put ownership in `session.Manager`. This works for session tools but does not cover other process-producing surfaces cleanly.

## Consequences

- The shared package becomes a cross-cutting dependency consumed by ACP, environment providers, hooks, and extension runtime.
- Tests must cover checkpoint durability, PID reuse detection, boot reconciliation, interrupt scoping, and cleanup behavior.
- The TechSpec must define package-boundary rules so lower-level packages do not import `daemon`.

## Implementation Notes

- Keep the package independent from daemon composition and session manager internals.
- Use `procutil.MatchesStartTime` where possible to avoid PID reuse mistakes.
- Treat interrupt as cooperative first, then escalate through process group signaling for owned local processes.

## References

- `.compozy/tasks/hermes/analysis/analysis_tools_security.md`
- Issues: 29, 30
