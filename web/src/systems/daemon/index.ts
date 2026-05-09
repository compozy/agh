// Types
export type {
  DaemonStatusPayload,
  DaemonStatusResponse,
  HealthPayload,
  MemoryHealthPayload,
  ObserveHealthResponse,
} from "./types";

// Adapters
export { fetchDaemonStatus, fetchHealth } from "./adapters/daemon-api";

// Query infrastructure
export { daemonKeys } from "./lib/query-keys";
export { daemonHealthOptions, daemonStatusOptions } from "./lib/query-options";

// Hooks
export {
  deriveDaemonConnectionStatus,
  useDaemonConnectionStatus,
  type ConnectionStatus,
} from "./hooks/use-daemon-connection-status";
export { useDaemonHealth } from "./hooks/use-daemon-health";
export { useDaemonStatus } from "./hooks/use-daemon-status";
