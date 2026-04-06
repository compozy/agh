# Restore Real Driver Boot, Background Daemon UX, and Truthful Dashboard State

## Summary

- Make `agh start` detach by default, keep `--foreground` for terminal-attached debugging, and always persist logs to `~/.agh/logs/agh.log`.
- Replace placeholder production driver boot with real driver instantiation from `[runtime.drivers.*]`, and fail `session start` immediately when bootstrap agents cannot launch a real PTY-backed process.
- Make the dashboard truthful and stable: show friendly IDE labels plus model, surface load/bootstrap errors, and stop the canvas from refitting during normal zoom/pan activity.

## Implementation Scope

### Daemon lifecycle and CLI contract

- Change `agh start` to detach by default and return after the child daemon is ready.
- Add `agh start --foreground` to keep the current terminal-attached behavior for debugging.
- Add an internal daemon-only entrypoint used by the detached launcher.
- Extend status output with the persistent log path and print PID/socket/dashboard/log path after detached start succeeds.

### Persistent logging and error surfacing

- Extend AGH home layout with `~/.agh/logs/agh.log`.
- Write daemon logs to the file always, and to stderr only in foreground mode.
- Log bootstrap failures, driver instantiation failures, and dashboard/websocket failures with structured fields.
- Ensure `agh session start` returns the real bootstrap failure instead of leaving a fake-active session.

### Real driver registration from runtime config

- Instantiate concrete drivers from `config.Runtime.Drivers`, honoring configured `binary` paths and OpenCode `mode`.
- Keep `kernel.WithDriver(...)` as the override path for tests/custom injection.
- Remove the bootstrap special case that treated `ErrNotImplemented` as success.
- Mark bootstrap agents ready only after a real process is attached and readiness succeeds.
- Send an explicit kickoff message to the supervisor once ready; advisor remains passive until messaged.

### Claude delivery parity with the POC

- Replace newline-only PTY writes with composer-style input delivery: literal segments, `Ctrl+J` for embedded newlines, final `Enter` to submit.
- Normalize CRLF to LF before delivery.

### Dashboard API/types and canvas behavior

- Add a friendly `driver_label` field to dashboard-facing agent payloads while keeping raw `driver`.
- Render agent headers as friendly IDE label plus model.
- Surface topology/bootstrap/load errors explicitly in the dashboard.
- Auto-fit only on initial load, explicit fit actions, or topology shape changes.
- Preserve viewport during status-only updates and active navigation.
- Add zoom presets `1`–`9` and double-click background fit.

## Verification

- Add/adjust Go unit and integration tests for detached/foreground start, status/log-path output, runtime-configured driver registration, bootstrap failure rollback, supervisor kickoff, and Claude multiline delivery.
- Add/adjust frontend tests for friendly labels, viewport stability, shortcut behavior, and explicit dashboard error states.
- Run `make verify` before claiming completion.
