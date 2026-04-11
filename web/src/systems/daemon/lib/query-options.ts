import { queryOptions } from "@tanstack/react-query";

import { fetchDaemonStatus, fetchHealth } from "../adapters/daemon-api";
import { daemonKeys } from "./query-keys";

export function daemonHealthOptions() {
  return queryOptions({
    queryKey: daemonKeys.health(),
    queryFn: ({ signal }) => fetchHealth(signal),
    refetchInterval: 10_000,
    retry: 1,
    staleTime: 5_000,
  });
}

export function daemonStatusOptions() {
  return queryOptions({
    queryKey: daemonKeys.status(),
    queryFn: ({ signal }) => fetchDaemonStatus(signal),
    retry: 1,
    staleTime: 60_000,
  });
}
