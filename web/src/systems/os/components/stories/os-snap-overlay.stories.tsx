import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useRef, useState } from "react";

import { deriveSnapRect } from "../../lib/os-snap-geometry";
import { claimedHalf } from "../../lib/os-snap-window-targets";
import { OS_SNAP_ZONES } from "../../lib/os-snap-zones";
import type { OsDesktopBounds, OsSnapZoneId } from "../../lib/os-types";
import { OsSnapOverlaySheet, type OsSnapOverlayState } from "../os-snap-overlay";
import { OsWindowFrame } from "../os-window-frame";
import { buildDeskItems, DesktopShell } from "./_desktop";

const meta: Meta<typeof OsSnapOverlaySheet> = {
  title: "systems/os/components/OsSnapOverlay",
  component: OsSnapOverlaySheet,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "FancyZones-class drop overlay (ADR-009 constants contract): 80ms linear fade-in, 150ms ease-out target morph on insets/background/border/opacity only, backdrop blur on the active target only, dashed tokenized accent outline, full collapse under reduced motion. States: idle (no zone captured — nothing renders), eligible (quiet outline), active (release target).",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/** A window silhouette mid-drag, so captures show the overlay in context. */
function DraggedWindow({ rect }: { rect: { x: number; y: number; w: number; h: number } }) {
  return (
    <div
      className="absolute"
      style={{ left: rect.x, top: rect.y, width: rect.w, height: rect.h, zIndex: 3 }}
    >
      <OsWindowFrame title="Tasks" focused className="h-full w-full">
        <div className="flex flex-1 items-center justify-center text-small-body text-subtle">
          Dragging toward a zone…
        </div>
      </OsWindowFrame>
    </div>
  );
}

/**
 * Measures the live win-layer box and derives the preview through the REAL
 * production math (`deriveSnapRect`), so the story exercises the same
 * derivation clients run.
 */
function SnapScene({
  zoneId,
  state,
  reducedMotion = false,
  windowRect,
}: {
  zoneId: OsSnapZoneId | null;
  state: OsSnapOverlayState;
  reducedMotion?: boolean;
  windowRect: { x: number; y: number; w: number; h: number };
}) {
  const layerRef = useRef<HTMLDivElement | null>(null);
  const [bounds, setBounds] = useState<OsDesktopBounds | null>(null);

  useEffect(() => {
    const layer = layerRef.current;
    if (!layer) return;
    const observer = new ResizeObserver(entries => {
      const entry = entries[0];
      if (!entry) return;
      setBounds({ width: entry.contentRect.width, height: entry.contentRect.height });
    });
    observer.observe(layer);
    return () => observer.disconnect();
  }, []);

  return (
    <DesktopShell dockItems={buildDeskItems({ open: ["tasks"] })}>
      <div ref={layerRef} data-slot="os-win-layer" className="absolute inset-0">
        <DraggedWindow rect={windowRect} />
        {zoneId !== null && bounds !== null ? (
          <OsSnapOverlaySheet
            rect={deriveSnapRect(OS_SNAP_ZONES[zoneId], bounds)}
            bounds={bounds}
            state={state}
            reducedMotion={reducedMotion}
            // Production layering: the held window paints above the sheet.
            zIndex={2}
          />
        ) : null}
      </div>
    </DesktopShell>
  );
}

/** Pointer away from every edge: no zone captured, no affordance renders. */
export const Idle: Story = {
  render: () => (
    <SnapScene zoneId={null} state="active" windowRect={{ x: 420, y: 180, w: 560, h: 400 }} />
  ),
};

/** Quiet dashed accent outline — a captured zone that is not the release target. */
export const EligibleLeftHalf: Story = {
  render: () => (
    <SnapScene zoneId="left" state="eligible" windowRect={{ x: 120, y: 160, w: 560, h: 400 }} />
  ),
};

/** The release target: accent-lit sheet, backdrop blur on this target only. */
export const ActiveLeftHalf: Story = {
  render: () => (
    <SnapScene zoneId="left" state="active" windowRect={{ x: 60, y: 140, w: 560, h: 400 }} />
  ),
};

/** Corner quarter target (top edge alone stays unbound — zoom owns full). */
export const ActiveTopRightQuarter: Story = {
  render: () => (
    <SnapScene zoneId="top-right" state="active" windowRect={{ x: 640, y: 90, w: 560, h: 400 }} />
  ),
};

/** Reduced motion (system pref or in-product toggle): no fade, no morph — the affordance itself never disappears (US-021.EC-4). */
export const ReducedMotion: Story = {
  render: () => (
    <SnapScene
      zoneId="left"
      state="active"
      reducedMotion
      windowRect={{ x: 60, y: 140, w: 560, h: 400 }}
    />
  ),
};

/**
 * Window-relative split target: dragging over an existing window's outer
 * third highlights the claimed half of THAT window (gutter-split) instead of
 * a desktop-edge zone — the same sheet, a window-sized rect.
 */
export const WindowSplitTarget: Story = {
  render: () => <WindowSplitScene />,
};

function WindowSplitScene() {
  const target = { x: 620, y: 60, w: 560, h: 560 };
  const layerRef = useRef<HTMLDivElement | null>(null);
  const [bounds, setBounds] = useState<OsDesktopBounds | null>(null);

  useEffect(() => {
    const layer = layerRef.current;
    if (!layer) return;
    const observer = new ResizeObserver(entries => {
      const entry = entries[0];
      if (!entry) return;
      setBounds({ width: entry.contentRect.width, height: entry.contentRect.height });
    });
    observer.observe(layer);
    return () => observer.disconnect();
  }, []);

  return (
    <DesktopShell dockItems={buildDeskItems({ open: ["tasks", "vault"] })}>
      <div ref={layerRef} data-slot="os-win-layer" className="absolute inset-0">
        <div
          className="absolute"
          style={{ left: target.x, top: target.y, width: target.w, height: target.h, zIndex: 1 }}
        >
          <OsWindowFrame title="Vault" focused={false} className="h-full w-full">
            <div className="flex flex-1 items-center justify-center text-small-body text-subtle">
              Split target
            </div>
          </OsWindowFrame>
        </div>
        <DraggedWindow rect={{ x: 240, y: 320, w: 520, h: 380 }} />
        {bounds !== null ? (
          <OsSnapOverlaySheet
            rect={claimedHalf(target, "bottom")}
            bounds={bounds}
            state="active"
            zIndex={2}
          />
        ) : null}
      </div>
    </DesktopShell>
  );
}
