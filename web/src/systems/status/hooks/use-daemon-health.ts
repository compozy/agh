import { useQuery } from "@tanstack/react-query";
import type { ConnectionStatus } from "@agh/ui";

import type { HealthPayload } from "../types";
import { daemonHealthOptions } from "../lib/query-options";
import { deriveDaemonConnectionStatus } from "./use-daemon-connection-status";

interface DaemonHealthResult {
  health: HealthPayload | undefined;
  connectionStatus: ConnectionStatus;
  isInitialLoading: boolean;
}

export function useDaemonHealth(): DaemonHealthResult {
  const query = useQuery(daemonHealthOptions());

  return {
    health: query.data,
    connectionStatus: deriveDaemonConnectionStatus(query),
    isInitialLoading: query.isLoading,
  };
}
