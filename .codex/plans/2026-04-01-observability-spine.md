# AGH Agent-Native Observability Spine

## Summary

- Replace the current split model (`dashboard` live state, partial `session.db` audit, grep-only `slog`, ephemeral NATS/WebSocket flows) with one canonical observability spine that every runtime component writes to and every consumer reads from.
- Make the canonical source of truth local-first and daemon-owned: a global append-only operational ledger under `~/.agh/observability/`, not the dashboard and not OpenTelemetry directly.
- Keep privileged global visibility only for external operator agents and internal `supervisor`/`auditor` roles; normal worker/reviewer agents remain session/workgroup scoped.
- Enable full PTY audit by default: capture both PTY output and kernel-delivered input as rolling compressed transcript segments, indexed from the observability ledger.
- Treat OpenTelemetry as an optional export path, not the backend. Treat JetStream as a future option for distributed AGH, not the primary ledger in this phase.

## Key Changes

- Add `internal/observability` with:
  - `ObservationEvent` as the canonical runtime record: `event_id`, `seq`, `timestamp`, `observed_timestamp`, `severity`, `event_type`, `component`, `session_id/name`, `workgroup_id/name`, `agent_id/name`, `actor_type`, `actor_id`, `trace_id`, `span_id`, `correlation_id`, `payload`.
  - A single-writer `RuntimeStore` backed by `~/.agh/observability/runtime.db` for global events, client sessions, stream subscriptions, transcript indexes, and current projections.
  - A live `Broadcaster` that publishes only after durable append; reconnect/replay uses `after_seq` from `runtime.db`.
  - `TranscriptSink` that records PTY input/output frames to rolling compressed files under `~/.agh/observability/transcripts/{session}/{agent}/` and writes segment/frame metadata to `runtime.db`.
- Make all runtime mutations emit explicit domain events before/after state transitions. The mandatory v1 catalog is:
  - `daemon_started`, `daemon_stopped`, `session_started`, `session_stopped`, `session_resumed`
  - `workgroup_created`, `workgroup_state_changed`, `workgroup_destroyed`
  - `agent_spawned`, `agent_ready`, `agent_state_changed`, `agent_killed`, `agent_restarted`, `agent_health_failed`
  - `message_sent`, `message_broadcast`, `message_escalated`, `message_failed`
  - `hook_received`, `blackboard_appended`, `status_updated`, `scope_violation`, `limit_exceeded`
  - `pty_input_written`, `pty_output_chunk`, `pty_closed`
  - `dashboard_client_connected`, `dashboard_client_disconnected`, `topology_stream_connected`, `topology_stream_disconnected`, `pty_stream_connected`, `pty_stream_disconnected`
  - `api_request`, `cli_request`
- Bridge `slog` into the same spine using the existing `internal/logger.WithObserver` hook so logs become queryable correlated records instead of a separate blind file. Add `trace_id`, `span_id`, and `event_id` to session-scoped logs.
- Keep `session.db` for session-local blackboard/status compatibility, but stop treating it as the runtime audit source. Operational history lives in `runtime.db`; session views are projections.
- Extend PTY capture at the root cause point:
  - output capture from `internal/pty.Manager.readLoop`
  - input capture from `Process.Write` / driver message delivery path
  - every transcript frame indexed by `session`, `agent`, `direction`, `seq`, `byte_count`, `segment_id`, `offset`, `timestamp`
- Preserve current topology/dashboard behavior, but make the dashboard a consumer of the spine:
  - add an audit timeline pane, live client/stream panel, and transcript replay entrypoint
  - restore an explicit multi-session selector so the UI matches the documented global-dashboard contract
  - stop relying on hidden UI-only state as the only live runtime view
- Do not introduce JetStream as the primary ledger in this phase. If needed later for multi-daemon replay/fanout, layer it behind the same `ObservationEvent` contract.
- Add config:
  - `[observability] enabled`, `retention_days`, `max_global_bytes`, `stream_buffer`
  - `[observability.transcripts] enabled`, `capture_input`, `capture_output`, `segment_bytes`, `max_bytes_per_session`
  - `[observability.otlp] enabled`, `endpoint`, `protocol`, `headers`

## Public Interfaces / Types

- Add CLI namespace `agh observe` and keep current commands as compatibility aliases where sensible:
  - `agh observe sessions`
  - `agh observe events [--session --workgroup --agent --type --since --after-seq --follow]`
  - `agh observe transcript <agent> [--session --since --after-seq --follow]`
  - `agh observe streams [--session]`
  - `agh observe health`
- Add privileged API surfaces over the existing daemon HTTP server:
  - `GET /api/observability/sessions`
  - `GET /api/observability/events`
  - `GET /api/observability/events/stream` as SSE/NDJSON
  - `GET /api/observability/transcripts/:session/:agent`
  - `GET /api/observability/streams`
  - `GET /api/observability/clients`
- Authorization model:
  - external operator agents using the local daemon/CLI are privileged by default on the host
  - internal AGH agents get global observability only if role type is `supervisor` or `auditor`
  - existing workgroup/session scoping remains the default for all other roles

## Test Plan

- Add an event coverage matrix test that fails if any required lifecycle path does not emit its canonical event.
- Add end-to-end kernel tests covering: session start/resume/stop, workgroup create/destroy, agent spawn/ready/kill, hook ingestion, message/broadcast/escalate, blackboard/status writes, scope violations, dashboard client connect/disconnect.
- Add transcript tests covering PTY input/output capture, segment rotation, replay by `after_seq`, reconnect after daemon restart, and bounded retention cleanup.
- Add privileged-access tests proving supervisors/auditors can query globally while workers remain scoped.
- Add compatibility tests for existing `agh events`, `agh ps`, `agh topology`, and `agh dashboard` behavior against the new projections.
- Run full verification gate after implementation: `make verify`.

## Assumptions and Defaults

- Preserve the repository’s single-binary, local-first architecture; no mandatory sidecar collector or external backend in v1.
- Full PTY transcript capture is enabled by default with rolling compressed segments and retention controls.
- OpenTelemetry export is optional and off by default; it exists to export the same canonical events/signals, not to replace the local ledger.
- SQLite-backed `runtime.db` is the primary operational ledger because AGH is still single-daemon/local-first; JetStream is deferred until AGH needs durable multi-process or multi-node observability fanout.
- Primary external references informing these decisions:
  - OpenTelemetry is instrumentation/export infrastructure and “not an observability backend itself”: https://opentelemetry.io/docs/what-is-opentelemetry/
  - OpenTelemetry Collector unifies receivers/processors/exporters for logs, metrics, and traces: https://opentelemetry.io/docs/collector/ and https://opentelemetry.io/docs/collector/configuration/
  - OpenTelemetry log records support `Timestamp`, `ObservedTimestamp`, `TraceId`, `SpanId`, severity, body, and attributes: https://opentelemetry.io/es/docs/concepts/signals/logs/
  - NATS Core is at-most-once, while JetStream consumers add replay and at-least-once semantics: https://docs.nats.io/nats-concepts/jetstream/consumers and https://docs.nats.io/nats-concepts/jetstream/streams
