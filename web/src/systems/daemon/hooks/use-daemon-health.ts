import { useQuery } from "@tanstack/react-query";

import type { ConnectionStatus } from "@/components/connection-indicator";
import type { HealthPayload } from "../types";
import { daemonHealthOptions } from "../lib/query-options";

interface DaemonHealthResult {
  health: HealthPayload | undefined;
  connectionStatus: ConnectionStatus;
  isInitialLoading: boolean;
}

export function useDaemonHealth(): DaemonHealthResult {
  const query = useQuery(daemonHealthOptions());

  let connectionStatus: ConnectionStatus;
  if (query.isPending || (query.isFetching && !query.data)) {
    connectionStatus = "reconnecting";
  } else if (query.isSuccess) {
    connectionStatus = "connected";
  } else if (query.isFetching && query.isError) {
    connectionStatus = "reconnecting";
  } else {
    connectionStatus = "disconnected";
  }

  return {
    health: query.data,
    connectionStatus,
    isInitialLoading: query.isLoading,
  };
}
