import { useRef, type KeyboardEvent, type PointerEvent } from "react";

import { cn } from "@agh/ui";

import { useDesktop } from "../hooks/use-desktop";
import { useOsShell } from "../hooks/use-os-shell";
import type { ProjectedSeam } from "../lib/window-manager-types";

function seamSpan(seam: ProjectedSeam, width: number, height: number): number {
  return seam.orientation === "vertical" ? width : height;
}

function resizeKeyDelta(event: KeyboardEvent<HTMLElement>): number | null {
  if (event.key === "ArrowLeft" || event.key === "ArrowUp") return -0.02;
  if (event.key === "ArrowRight" || event.key === "ArrowDown") return 0.02;
  return null;
}

function LayoutSeam({
  seam,
  width,
  height,
}: {
  seam: ProjectedSeam;
  width: number;
  height: number;
}) {
  const { manager } = useOsShell();
  const start = useRef<{ coordinate: number; pointerId: number } | null>(null);
  const vertical = seam.orientation === "vertical";

  const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    event.currentTarget.setPointerCapture(event.pointerId);
    start.current = {
      coordinate: vertical ? event.clientX : event.clientY,
      pointerId: event.pointerId,
    };
  };
  const handlePointerUp = (event: PointerEvent<HTMLDivElement>) => {
    const gesture = start.current;
    start.current = null;
    if (!gesture || gesture.pointerId !== event.pointerId) return;
    const deltaPixels = (vertical ? event.clientX : event.clientY) - gesture.coordinate;
    const span = seamSpan(seam, width, height);
    if (span > 0) {
      manager.getState().resizeLayout(seam.splitId, seam.boundaryIndex, deltaPixels / span);
    }
  };

  return (
    <div
      role="separator"
      tabIndex={0}
      aria-label={`Resize boundary ${seam.boundaryIndex + 1}`}
      aria-orientation={vertical ? "vertical" : "horizontal"}
      aria-valuemin={Math.round(seam.minValue)}
      aria-valuemax={Math.round(seam.maxValue)}
      aria-valuenow={Math.round(seam.value)}
      data-split-id={seam.splitId}
      data-boundary-index={seam.boundaryIndex}
      className={cn(
        "absolute z-30 touch-none rounded-pill outline-none",
        "before:absolute before:rounded-pill before:bg-line-strong",
        "hover:before:bg-accent focus-visible:shadow-focus-ring focus-visible:before:bg-accent",
        vertical
          ? "w-3 cursor-col-resize before:top-0 before:bottom-0 before:left-1/2 before:w-px before:-translate-x-1/2"
          : "h-3 cursor-row-resize before:top-1/2 before:right-0 before:left-0 before:h-px before:-translate-y-1/2"
      )}
      style={{
        left: seam.rect.x - (vertical ? 6 : 0),
        top: seam.rect.y - (vertical ? 0 : 6),
        width: vertical ? 12 : seam.rect.w,
        height: vertical ? seam.rect.h : 12,
      }}
      onPointerDown={handlePointerDown}
      onPointerCancel={() => {
        start.current = null;
      }}
      onPointerUp={handlePointerUp}
      onKeyDown={event => {
        const delta = resizeKeyDelta(event);
        if (delta === null) return;
        event.preventDefault();
        manager.getState().resizeLayout(seam.splitId, seam.boundaryIndex, delta);
      }}
    />
  );
}

/** Structural seams come directly from split IDs and boundary indexes. */
export function OsSnapSeamLayer() {
  const projection = useDesktop(state =>
    state.activeDesktopId ? state.projections[state.activeDesktopId] : undefined
  );
  if (!projection) return null;
  return (
    <>
      {projection.seams.map(seam => (
        <LayoutSeam
          key={seam.id}
          seam={seam}
          width={projection.workArea.w}
          height={projection.workArea.h}
        />
      ))}
    </>
  );
}
