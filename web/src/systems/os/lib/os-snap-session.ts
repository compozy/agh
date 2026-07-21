import { snapWorkArea } from "./os-snap-geometry";
import { resolveSnapZone, type OsSnapPoint } from "./os-snap-zones";
import type { OsDesktopBounds, OsSnapZoneId } from "./os-types";

/**
 * One zone-tracking session per drag gesture (Hermes drag-session pattern):
 * geometry snapshotted once at start, raw moves rAF-coalesced so hit testing
 * and hint publishing happen at most once per frame, Esc aborts synchronously
 * and the drop lands at the final pointer position.
 */

export interface SnapZoneSessionOptions {
  /** Win-layer box at drag start; client points translate against it. */
  layer: { left: number; top: number };
  /** Win-layer bounds at drag start; the work area derives from them. */
  bounds: OsDesktopBounds;
  /** Hint sink — called only when the resolved zone changes. */
  publish(zoneId: OsSnapZoneId | null): void;
  /** Per-frame layer-space pointer callback (the detached drag follows it). */
  onFrame?(point: OsSnapPoint): void;
}

export interface SnapZoneSession {
  /** Feeds a client-coordinate pointer sample; processing is per-frame. */
  move(clientX: number, clientY: number): void;
  /** Flushes the pending sample and returns the zone under the final point. */
  end(): OsSnapZoneId | null;
  /** Cancels: clears the hint, drops pending work, ignores further moves. */
  abort(): void;
  aborted(): boolean;
}

export function createSnapZoneSession(options: SnapZoneSessionOptions): SnapZoneSession {
  const { layer, publish, onFrame } = options;
  const area = snapWorkArea(options.bounds);
  let pending: OsSnapPoint | null = null;
  let raf = 0;
  let zone: OsSnapZoneId | null = null;
  let aborted = false;

  const flush = () => {
    raf = 0;
    if (aborted || pending === null) return;
    const point = pending;
    pending = null;
    onFrame?.(point);
    const next = resolveSnapZone(point, area);
    if (next !== zone) {
      zone = next;
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
      const final = aborted ? null : zone;
      zone = null;
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
      zone = null;
      publish(null);
    },
    aborted: () => aborted,
  };
}
