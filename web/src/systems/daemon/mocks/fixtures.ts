import type { DaemonStatusPayload, HealthPayload } from "../types";

export const daemonHealthFixture: HealthPayload = {
  status: "ok",
  uptime_seconds: 7_200,
  active_sessions: 3,
  active_agents: 5,
  bridges: {
    total_instances: 2,
    route_count: 4,
    delivery_backlog: 1,
    delivery_dropped_total: 0,
    delivery_failures_total: 0,
    auth_failures_total: 0,
    status_counts: {
      disabled: 0,
      starting: 0,
      ready: 2,
      degraded: 0,
      auth_required: 0,
      error: 0,
    },
  },
  global_db_size_bytes: 1_048_576,
  session_db_size_bytes: 786_432,
  persistence: {
    status: "ok",
    global_db_size_bytes: 1_048_576,
    session_db_size_bytes: 786_432,
  },
  retention: {
    enabled: true,
    retention_days: 7,
    sweep_interval_seconds: 86_400,
    last_sweep_status: "ok",
    last_sweep_at: "2026-04-17T18:00:00Z",
    last_cutoff_at: "2026-04-10T18:00:00Z",
    deleted_event_summaries: 0,
    deleted_token_stats: 0,
    deleted_permission_log_rows: 0,
  },
  failures: {
    status: "ok",
    total: 0,
  },
  agent_probes: [
    {
      agent_name: "coder",
      provider: "claude",
      command: "claude --acp",
      executable: "/usr/local/bin/claude",
      status: "ok",
      checked_at: "2026-04-17T18:00:00Z",
      duration_ms: 12,
    },
  ],
  version: "0.1.0-storybook",
};

export const daemonStatusFixture: DaemonStatusPayload = {
  status: "running",
  pid: 4242,
  started_at: "2026-04-17T18:00:00Z",
  socket: "/tmp/agh.sock",
  http_host: "127.0.0.1",
  http_port: 2123,
  user_home_dir: "/Users/pedro",
  active_sessions: 3,
  total_sessions: 11,
  version: "0.1.0-storybook",
};
