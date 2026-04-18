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
