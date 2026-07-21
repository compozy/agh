import { cn } from "@/lib/utils";

import { useDesktop } from "../hooks/use-desktop";
import { useOsReducedMotion } from "../hooks/use-os-reduced-motion";
import { deriveSnapRect } from "../lib/os-snap-geometry";
import { OS_SNAP_ZONES } from "../lib/os-snap-zones";
import type { OsDesktopBounds, OsRect } from "../lib/os-types";

/**
 * FancyZones-class drop overlay (ADR-009 motion contract, Hermes provenance):
 * the container fades in at 80ms linear; ONE dashed accent sheet morphs
 * between targets at 150ms ease-out transitioning only insets/background/
 * border/opacity (never `transition-all` — blur interpolation is the most
 * expensive drag paint); backdrop blur applies to the active target only;
 * reduced motion (system preference or in-product toggle) collapses fade and
 * morph while the affordance itself stays visible (US-021.EC-4).
 */

/** Sheet inset from the zone edge (px), so adjacent zone previews read apart. */
const SHEET_PAD = 6;
/** Fallback stack position when no dragged window anchors the sheet. */
const OVERLAY_Z = 9999;

export type OsSnapOverlayState = "eligible" | "active";

export interface OsSnapOverlaySheetProps {
  /** Preview box in win-layer coordinates (already derived and clamped). */
  rect: OsRect;
  /** Win-layer box the longhand insets resolve against. */
  bounds: OsDesktopBounds;
  /** `active` = the release target (accent-lit + blur); `eligible` = quiet outline. */
  state?: OsSnapOverlayState;
  reducedMotion?: boolean;
  /**
   * Stack position: just below the dragged window, above everything else —
   * the held window stays crisp while the blur fogs the zone under it.
   */
  zIndex?: number;
}

/** Presentational sheet — the story and the connected overlay share it. */
export function OsSnapOverlaySheet({
  rect,
  bounds,
  state = "active",
  reducedMotion = false,
  zIndex = OVERLAY_Z,
}: OsSnapOverlaySheetProps) {
  const active = state === "active";
  return (
    <div
      data-slot="os-snap-overlay"
      data-testid="os-snap-overlay"
      data-state={state}
      data-reduced-motion={reducedMotion ? "" : undefined}
      className="pointer-events-none absolute inset-0"
      style={{
        zIndex,
        animation: reducedMotion ? undefined : "os-snap-fade 80ms linear both",
      }}
    >
      <div
        data-slot="os-snap-overlay-sheet"
        className={cn(
          "absolute rounded-window border-2 border-dashed",
          !reducedMotion &&
            "transition-[top,right,bottom,left,background-color,border-color,opacity] duration-150 ease-out",
          // Blur is a static treatment, not motion — it survives reduced motion.
          active && "backdrop-blur-[2px]"
        )}
        style={{
          top: rect.y + SHEET_PAD,
          left: rect.x + SHEET_PAD,
          right: bounds.width - rect.x - rect.w + SHEET_PAD,
          bottom: bounds.height - rect.y - rect.h + SHEET_PAD,
          // Accent over an elevated wash so the fill dims content on the dark
          // canvas (a bare accent alpha disappears there) — Hermes formula on
          // our tokens.
          background: active
            ? "color-mix(in srgb, var(--color-accent) 18%, color-mix(in srgb, var(--color-elevated) 55%, transparent))"
            : "color-mix(in srgb, var(--color-accent) 5%, color-mix(in srgb, var(--color-elevated) 25%, transparent))",
          borderColor: `color-mix(in srgb, var(--color-accent) ${active ? 75 : 28}%, transparent)`,
        }}
      />
    </div>
  );
}

/**
 * Store-connected overlay: renders the active-target sheet for the single
 * published drag hint (one hint per frame — the resolver's contract).
 */
export function OsSnapOverlay() {
  const hint = useDesktop(desktop => desktop.snapHint);
  const bounds = useDesktop(desktop => desktop.desktopBounds);
  const draggedZ = useDesktop(desktop =>
    desktop.snapHint === null ? undefined : desktop.windows[desktop.snapHint.windowId]?.z
  );
  const reducedMotion = useOsReducedMotion();
  if (hint === null || bounds === null) return null;
  // Zone hints derive their preview rect; window-split hints carry the
  // claimed half computed at resolution time.
  const rect =
    hint.kind === "zone" ? deriveSnapRect(OS_SNAP_ZONES[hint.zoneId], bounds) : hint.rect;
  return (
    <OsSnapOverlaySheet
      rect={rect}
      bounds={bounds}
      state="active"
      reducedMotion={reducedMotion}
      zIndex={draggedZ === undefined ? OVERLAY_Z : Math.max(0, draggedZ - 1)}
    />
  );
}
