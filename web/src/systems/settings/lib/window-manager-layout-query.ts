import { queryOptions } from "@tanstack/react-query";

import { windowManagerSnapshotOptions } from "@/systems/os";

import { listWindowManagerLayoutProfiles } from "../adapters/window-manager-layouts-api";
import { settingsKeys } from "./query-keys";
import { windowManagerSnapshotToLayoutState } from "./window-manager-layout-projection";

export function windowManagerLayoutOptions(workspaceId: string) {
  const normalized = workspaceId.trim();
  const snapshot = windowManagerSnapshotOptions(normalized);
  return queryOptions({
    ...snapshot,
    select: windowManagerSnapshotToLayoutState,
  });
}

export function windowManagerLayoutProfilesOptions(workspaceId: string) {
  const normalized = workspaceId.trim();
  return queryOptions({
    queryKey: settingsKeys.windowManagerLayoutProfiles(normalized),
    queryFn: ({ signal }) => listWindowManagerLayoutProfiles(normalized, signal),
    enabled: normalized !== "",
    staleTime: 15_000,
  });
}
