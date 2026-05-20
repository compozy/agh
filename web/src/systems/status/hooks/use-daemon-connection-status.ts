import { useQuery } from "@tanstack/react-query";
import type { ConnectionStatus } from "@agh/ui";

import { daemonHealthOptions } from "../lib/query-options";

interface DaemonConnectionQueryState {
  data?: unknown;
  isError: boolean;
  isFetching: boolean;
  isPending: boolean;
  isSuccess: boolean;
}

function deriveDaemonConnectionStatus(query: DaemonConnectionQueryState): ConnectionStatus {
  if (query.isPending || (query.isFetching && query.data === undefined)) {
    return "connecting";
  }
  if (query.isSuccess) {
    return "connected";
  }
  if (query.isError) {
    return "error";
  }
  if (query.isFetching) {
    return "connecting";
  }
  return "disconnected";
}

function useDaemonConnectionStatus(): ConnectionStatus {
  return deriveDaemonConnectionStatus(useQuery(daemonHealthOptions()));
}

export { deriveDaemonConnectionStatus, useDaemonConnectionStatus };
export type { ConnectionStatus };
