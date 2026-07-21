import { cn } from "@/lib/utils";

import { useDesktop } from "../hooks/use-desktop";
import { useOsSeamDrag, type OsSeamDragModel } from "../hooks/use-os-seam-drag";
import { deriveSnapSeams, type OsSnapSeam } from "../lib/os-snap-seams";

/**
 * Linked-seam handles between adjacent snapped windows: each seam renders in
 * the shared gutter strip with the `ResizableHandle` visual grammar (hairline
 * strip, pill on hover/focus) and its own pointer/keyboard gesture — dragging
 * resizes both neighbors live (`seamPreview`), release commits fractions.
 * Seams derive from fractions each render; no group state exists to leak.
 */
export function OsSnapSeamLayer() {
  const windows = useDesktop(state => state.windows);
  const bounds = useDesktop(state => state.desktopBounds);
  const seamPreview = useDesktop(state => state.seamPreview);
  const presentation = useDesktop(state => state.presentation);
  const seamDrag = useOsSeamDrag();
  if (bounds === null || presentation === "compact") return null;
  const seams = deriveSnapSeams(Object.values(windows), bounds, seamPreview);
  if (seams.length === 0) return null;
  return (
    <>
      {seams.map(seam => (
        <OsSnapSeamHandle key={seam.id} seam={seam} seamDrag={seamDrag} />
      ))}
    </>
  );
}

function OsSnapSeamHandle({ seam, seamDrag }: { seam: OsSnapSeam; seamDrag: OsSeamDragModel }) {
  const vertical = seam.orientation === "vertical";
  return (
    <div
      role="separator"
      tabIndex={0}
      aria-orientation={vertical ? "vertical" : "horizontal"}
      aria-label="Resize snapped windows"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(seam.value * 100)}
      data-slot="os-snap-seam"
      data-testid={`os-snap-seam-${seam.id}`}
      className={cn(
        "group absolute flex touch-none items-center justify-center outline-none",
        "focus-visible:shadow-focus-ring",
        vertical ? "cursor-col-resize" : "cursor-row-resize"
      )}
      style={{
        left: seam.rect.x,
        top: seam.rect.y,
        width: seam.rect.w,
        height: seam.rect.h,
        zIndex: seam.z,
      }}
      onPointerDown={event => seamDrag.onSeamPointerDown(seam, event)}
      onKeyDown={event => seamDrag.onSeamKeyDown(seam, event)}
    >
      <div
        className={cn(
          "rounded-full bg-line-strong opacity-0 transition-opacity duration-fast ease-out",
          "group-hover:opacity-100 group-focus-visible:opacity-100",
          vertical ? "h-6 w-1" : "h-1 w-6"
        )}
      />
    </div>
  );
}
