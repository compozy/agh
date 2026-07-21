import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import { decodeDesktopEntry, decodeWindowEntry } from "./os-state-payloads";
import type { OsStateEntry, OsWallpaper, OsWindow } from "./os-types";

/**
 * One-shot HTTP read of a workspace's persisted desktop state — the Spaces
 * overview's data source for NON-active workspaces (the active space reads
 * the live WM store instead). Live sync stays with `OsStateClient`.
 */

export class DesktopStateApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "DesktopStateApiError";
  }
}

export async function fetchWorkspaceDesktopState(
  workspaceId: string,
  signal?: AbortSignal
): Promise<OsStateEntry[]> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/desktop-state",
    { params: { path: { workspace_id: workspaceId } }, signal }
  );
  if (apiRequestFailed(response, error)) {
    throw new DesktopStateApiError(
      defaultApiErrorMessage("Failed to load desktop state", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to load desktop state").entries;
}

export interface OsSpaceArrangement {
  /** Live (non-minimized) windows, z-ascending; unreadable entries dropped. */
  windows: OsWindow[];
  wallpaper: OsWallpaper;
}

/** Projects raw desktop-state entries into the overview's arrangement shape. */
export function decodeSpaceArrangement(entries: OsStateEntry[]): OsSpaceArrangement {
  const windows: OsWindow[] = [];
  let wallpaper: OsWallpaper = "ember";
  for (const entry of entries) {
    const desktop = decodeDesktopEntry(entry);
    if (desktop) {
      wallpaper = desktop.wallpaper;
      continue;
    }
    const win = decodeWindowEntry(entry);
    if (win && !win.minimized) windows.push(win);
  }
  windows.sort((a, b) => a.z - b.z);
  return { windows, wallpaper };
}
