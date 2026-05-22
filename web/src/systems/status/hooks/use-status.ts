import { useQuery } from "@tanstack/react-query";

import { daemonStatusOptions } from "../lib/query-options";

export function useStatus() {
  return useQuery(daemonStatusOptions());
}
