import { deriveSnapRect } from "./os-snap-geometry";
import type { OsWindowTargetCandidate } from "./os-snap-window-targets";
import type { OsDesktopRuntimeStore } from "./os-types";

/**
 * Split-candidate snapshot for a drag session (Hermes discipline: rects are
 * captured once at drag start). Every visible, non-maximized window except
 * the dragged one participates — snapped candidates use their derived rect,
 * floating ones their committed rect.
 */
export function snapTargetCandidates(
  state: Pick<OsDesktopRuntimeStore, "windows" | "desktopBounds">,
  draggedId: string
): OsWindowTargetCandidate[] {
  const bounds = state.desktopBounds;
  const candidates: OsWindowTargetCandidate[] = [];
  for (const win of Object.values(state.windows)) {
    if (win.id === draggedId || win.minimized || win.maximized) continue;
    const rect = win.snap !== null && bounds !== null ? deriveSnapRect(win.snap, bounds) : win.rect;
    candidates.push({ id: win.id, rect, z: win.z });
  }
  return candidates;
}
