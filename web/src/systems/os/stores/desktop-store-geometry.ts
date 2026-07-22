import {
  OS_DESKTOP_GUTTERS,
  type OsDesktopBounds,
  type OsRect,
  type OsSnapHint,
  type OsWindow,
} from "../lib/os-types";

export function nextFocus(windows: Record<string, OsWindow>, excludeId: string): string | null {
  let best: OsWindow | null = null;
  for (const win of Object.values(windows)) {
    if (win.id === excludeId || win.minimized) continue;
    if (best === null || win.z > best.z) best = win;
  }
  return best?.id ?? null;
}

/** Prototype clamp (os-v2.js:145-152): head stays reachable above the dock. */
export function clampRect(rect: OsRect, bounds: OsDesktopBounds): OsRect {
  const { top, right, bottom, left } = OS_DESKTOP_GUTTERS;
  const w = Math.min(rect.w, Math.max(1, bounds.width - left - right - 8));
  const h = Math.min(rect.h, Math.max(1, bounds.height - 90));
  const x = Math.max(left, Math.min(rect.x, bounds.width - w - right));
  const y = Math.max(top, Math.min(rect.y, bounds.height - bottom));
  return { x, y, w, h };
}

export function sameRect(a: OsRect, b: OsRect): boolean {
  return a.x === b.x && a.y === b.y && a.w === b.w && a.h === b.h;
}

export function sameSnapHint(a: OsSnapHint, b: OsSnapHint): boolean {
  if (a.windowId !== b.windowId || a.kind !== b.kind) return false;
  if (a.kind === "zone" && b.kind === "zone") return a.zoneId === b.zoneId;
  if (a.kind === "window" && b.kind === "window") {
    // The claimed rect is a pure function of target + side within one drag —
    // identity rides on the discrete fields.
    return a.targetId === b.targetId && a.side === b.side;
  }
  return false;
}

export function sameLocation(a: OsWindow["location"], b: OsWindow["location"]): boolean {
  return a.pathname === b.pathname && JSON.stringify(a.search) === JSON.stringify(b.search);
}
