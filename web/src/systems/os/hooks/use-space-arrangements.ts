import { useQueries } from "@tanstack/react-query";

import {
  decodeSpaceArrangement,
  fetchWorkspaceDesktopState,
  type OsSpaceArrangement,
} from "../lib/desktop-state-api";
import { useDesktop } from "./use-desktop";

export interface OsSpaceCardModel {
  workspaceId: string;
  arrangement: OsSpaceArrangement | null;
  /** `loading` while a non-active space's persisted arrangement is in flight. */
  status: "live" | "loading" | "error";
}

/**
 * Per-workspace arrangements for the Spaces overview (US-010.AC-2): the
 * active space mirrors the live WM store; every other space reads its
 * persisted desktop state over HTTP while the overview is open.
 */
export function useSpaceArrangements(
  workspaceIds: string[],
  activeWorkspaceId: string | null,
  options: { enabled: boolean }
): Map<string, OsSpaceCardModel> {
  const liveWindows = useDesktop(state => state.windows);
  const liveWallpaper = useDesktop(state => state.wallpaper);
  const activeArrangement: OsSpaceArrangement = {
    windows: Object.values(liveWindows)
      .filter(win => !win.minimized)
      .sort((a, b) => a.z - b.z),
    wallpaper: liveWallpaper,
  };
  const inactiveIds = workspaceIds.filter(id => id !== activeWorkspaceId);
  const queries = useQueries({
    queries: inactiveIds.map(workspaceId => ({
      queryKey: ["os", "desktop-state", workspaceId] as const,
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        fetchWorkspaceDesktopState(workspaceId, signal),
      select: decodeSpaceArrangement,
      enabled: options.enabled,
      staleTime: 5_000,
    })),
  });

  const cards = new Map<string, OsSpaceCardModel>();
  if (activeWorkspaceId !== null) {
    cards.set(activeWorkspaceId, {
      workspaceId: activeWorkspaceId,
      arrangement: activeArrangement,
      status: "live",
    });
  }
  inactiveIds.forEach((workspaceId, index) => {
    const query = queries[index];
    cards.set(workspaceId, {
      workspaceId,
      arrangement: query.data ?? null,
      status: query.isError ? "error" : query.data ? "live" : "loading",
    });
  });
  return cards;
}
