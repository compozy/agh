import { Kbd } from "@agh/ui";

import { useOsWinLayer } from "../hooks/use-os-win-layer";
import { OsCompactStack } from "./os-compact-stack";
import { OsSnapOverlay } from "./os-snap-overlay";
import { OsSnapSeamLayer } from "./os-snap-seam";
import { OsWindow } from "./os-window";

/**
 * The desktop's window layer: floating windows over the wallpaper (or the
 * compact stack under 960px) and the empty-desk hint. Behavior (ids,
 * presentation, viewport re-clamp) lives in `useOsWinLayer`.
 */
export function OsWinLayer() {
  const { layerRef, windowIds, presentation, anyVisible } = useOsWinLayer();

  if (presentation === "compact") {
    return <OsCompactStack />;
  }

  return (
    <div ref={layerRef} data-slot="os-win-layer" className="absolute inset-0">
      {!anyVisible ? (
        <p
          data-testid="os-desk-hint"
          className="pointer-events-none absolute top-1/2 left-1/2 flex -translate-x-1/2 -translate-y-1/2 items-center gap-2 text-small-body text-subtle select-none"
        >
          <Kbd>⌘K</Kbd> to open anything — or pick a surface from the dock
        </p>
      ) : null}
      {windowIds.map(id => (
        <OsWindow key={id} windowId={id} />
      ))}
      <OsSnapSeamLayer />
      <OsSnapOverlay />
    </div>
  );
}
