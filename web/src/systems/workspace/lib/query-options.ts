import { queryOptions } from "@tanstack/react-query";

import { fetchWorkspaces } from "../adapters/workspace-api";
import { workspaceKeys } from "./query-keys";

export function workspacesListOptions() {
  return queryOptions({
    queryKey: workspaceKeys.list(),
    queryFn: ({ signal }) => fetchWorkspaces(signal),
    staleTime: 60_000,
  });
}
