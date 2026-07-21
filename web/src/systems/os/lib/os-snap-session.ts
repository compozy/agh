import { snapWorkArea } from "./os-snap-geometry";
import {
  resolveWindowTarget,
  type OsWindowTargetCandidate,
  type OsWindowTargetHit,
} from "./os-snap-window-targets";
import { resolveSnapZone, type OsSnapPoint } from "./os-snap-zones";
import type { OsDesktopBounds, OsSnapHint, OsSnapZoneId } from "./os-types";

/**
 * One zone-tracking session per drag gesture (Hermes drag-session pattern):
 * geometry snapshotted once at start, raw moves rAF-coalesced so hit testing
 * and hint publishing happen at most once per frame, Esc aborts synchronously
 * and the drop lands at the final pointer position. Desktop-edge bands
 * outrank window-relative split targets — a pointer inside a band never
 * offers a split.
 */

export type OsSnapResolution =
  | { kind: "zone"; zoneId: OsSnapZoneId }
  | { kind: "window"; target: OsWindowTargetHit };

/** Maps a session resolution onto the store's hint shape for `windowId`. */
export function resolutionToHint(
  windowId: string,
  res: OsSnapResolution | null
): OsSnapHint | null {
  if (res === null) return null;
  if (res.kind === "zone") return { windowId, kind: "zone", zoneId: res.zoneId };
  return {
    windowId,
    kind: "window",
    targetId: res.target.targetId,
    side: res.target.side,
    rect: res.target.rect,
  };
}

function sameResolution(a: OsSnapResolution | null, b: OsSnapResolution | null): boolean {
  if (a === null || b === null) return a === b;
  if (a.kind === "zone") return b.kind === "zone" && a.zoneId === b.zoneId;
  return (
    b.kind === "window" &&
    a.target.targetId === b.target.targetId &&
    a.target.side === b.target.side
  );
}

export interface SnapZoneSessionOptions {
  /** Win-layer box at drag start; client points translate against it. */
  layer: { left: number; top: number };
  /** Win-layer bounds at drag start; the work area derives from them. */
  bounds: OsDesktopBounds;
  /** Split candidates snapshotted at drag start (excludes the dragged window). */
  windowTargets?: OsWindowTargetCandidate[];
  /** Hint sink — called only when the resolution changes. */
  publish(resolution: OsSnapResolution | null): void;
  /** Per-frame layer-space pointer callback (the detached drag follows it). */
  onFrame?(point: OsSnapPoint): void;
}

export interface SnapZoneSession {
  /** Feeds a client-coordinate pointer sample; processing is per-frame. */
  move(clientX: number, clientY: number): void;
  /** Flushes the pending sample and returns the resolution under the final point. */
  end(): OsSnapResolution | null;
  /** Cancels: clears the hint, drops pending work, ignores further moves. */
  abort(): void;
  aborted(): boolean;
}

export function createSnapZoneSession(options: SnapZoneSessionOptions): SnapZoneSession {
  const { layer, publish, onFrame } = options;
  const area = snapWorkArea(options.bounds);
  const targets = options.windowTargets ?? [];
  let pending: OsSnapPoint | null = null;
  let raf = 0;
  let current: OsSnapResolution | null = null;
  let aborted = false;

  const flush = () => {
    raf = 0;
    if (aborted || pending === null) return;
    const point = pending;
    pending = null;
    onFrame?.(point);
    // Edge bands first (with exit hysteresis on the active zone); window
    // targets only arm away from every band.
    const activeZone = current?.kind === "zone" ? current.zoneId : null;
    const zoneId = resolveSnapZone(point, area, activeZone);
    const next: OsSnapResolution | null =
      zoneId !== null
        ? { kind: "zone", zoneId }
        : (target => (target === null ? null : { kind: "window" as const, target }))(
            resolveWindowTarget(point, targets)
          );
    if (!sameResolution(next, current)) {
      current = next;
      publish(next);
    }
  };

  return {
    move(clientX, clientY) {
      if (aborted) return;
      pending = { x: clientX - layer.left, y: clientY - layer.top };
      raf ||= requestAnimationFrame(flush);
    },
    end() {
      if (raf) {
        cancelAnimationFrame(raf);
        raf = 0;
      }
      flush();
      const final = aborted ? null : current;
      current = null;
      publish(null);
      return final;
    },
    abort() {
      if (aborted) return;
      aborted = true;
      if (raf) {
        cancelAnimationFrame(raf);
        raf = 0;
      }
      pending = null;
      current = null;
      publish(null);
    },
    aborted: () => aborted,
  };
}
