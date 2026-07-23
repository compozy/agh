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

export function windowManagerLayoutProfilesOptions() {
  return queryOptions({
    queryKey: settingsKeys.windowManagerLayoutProfiles(),
    queryFn: ({ signal }) => listWindowManagerLayoutProfiles(signal),
    staleTime: 15_000,
  });
}
