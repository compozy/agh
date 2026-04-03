import { queryOptions } from "@tanstack/react-query";

import { fetchHealth } from "../adapters/daemon-api";
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
