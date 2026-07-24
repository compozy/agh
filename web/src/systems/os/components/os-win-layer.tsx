import { Kbd } from "@agh/ui";

import type { DesktopLayerModel, OsWinLayerModel } from "../hooks/use-os-win-layer";
import type { SnapTarget } from "../lib/snap-targets";
import type { LayoutProjection } from "../lib/window-manager-types";
import type { DesktopTransitionIntent } from "../stores/window-manager-store";
import { OsSnapOverlay } from "./os-snap-overlay";
import { OsSnapSeamLayer } from "./os-snap-seam";
import { OsWindow } from "./os-window";

function DesktopLayer({
  model,
  compact,
  reducedMotion,
  viewportReady,
  transition,
  onTransitionComplete,
  seamProjection,
  preview,
  onResizeLayout,
}: {
  model: DesktopLayerModel;
  compact: boolean;
  reducedMotion: boolean;
  viewportReady: boolean;
  transition: DesktopTransitionIntent | null;
  onTransitionComplete: () => void;
  seamProjection: LayoutProjection | undefined;
  preview: SnapTarget | null;
  onResizeLayout: (splitId: string, boundaryIndex: number, delta: number) => void;
}) {
  const incoming = transition?.toDesktopId === model.desktop.id;
  const outgoing = transition?.fromDesktopId === model.desktop.id;
  const transitionActive =
    transition !== null && transition.mode !== "instant" && (incoming || outgoing);
  const visible = viewportReady && (model.active || transitionActive);
  const interactive = viewportReady && model.active;
  const slideOffset =
    transition?.direction === "later" ? (incoming ? "4%" : "-4%") : incoming ? "-4%" : "4%";
  const transform =
    transitionActive && transition.mode === "slide" && !model.active
      ? `translateX(${slideOffset})`
      : "translateX(0)";
  const opacity = model.active ? 1 : outgoing && transition?.mode === "slide" ? 1 : 0;

  return (
    <section
      data-screen-label={`Desktop ${model.desktop.order + 1}: ${model.desktop.name}`}
      data-desktop-id={model.desktop.id}
      data-active={model.active ? "true" : "false"}
      aria-hidden={!interactive}
      inert={interactive ? undefined : true}
      className="absolute inset-0"
      onTransitionEnd={event => {
        if (
          event.target === event.currentTarget &&
          model.active &&
          incoming &&
          (event.propertyName === "opacity" || event.propertyName === "transform")
        ) {
          onTransitionComplete();
        }
      }}
      style={{
        contain: "strict",
        contentVisibility: visible ? "visible" : "hidden",
        opacity,
        pointerEvents: interactive ? "auto" : "none",
        transform,
        transition:
          reducedMotion || transition?.mode === "instant"
            ? "none"
            : "opacity var(--duration-shell-fast) ease-out, transform var(--duration-shell-fast) ease-out",
      }}
    >
      {interactive && !model.anyVisible ? (
        <p
          data-testid="os-desk-hint"
          className="pointer-events-none absolute top-1/2 left-1/2 flex -translate-x-1/2 -translate-y-1/2 items-center gap-2 text-small-body text-subtle select-none"
        >
          <Kbd>⌘K</Kbd> to open anything — or pick a surface from the dock
        </p>
      ) : null}
      {model.windowIds.map(id => (
        <OsWindow key={id} windowId={id} />
      ))}
      {!compact && interactive ? (
        <>
          <OsSnapSeamLayer projection={seamProjection} onResize={onResizeLayout} />
          <OsSnapOverlay preview={preview} />
        </>
      ) : null}
    </section>
  );
}

/** Every desktop tree remains mounted; only the client-active tree is interactive. */
export function OsWinLayer({
  model,
  reducedMotion,
  transition,
  preview,
  onTransitionComplete,
  onResizeLayout,
}: {
  model: OsWinLayerModel;
  reducedMotion: boolean;
  transition: DesktopTransitionIntent | null;
  preview: SnapTarget | null;
  onTransitionComplete: () => void;
  onResizeLayout: (splitId: string, boundaryIndex: number, delta: number) => void;
}) {
  const { layerRef, desktops, presentation, viewportState, activeProjection } = model;
  return (
    <div ref={layerRef} data-slot="os-win-layer" className="absolute inset-0">
      {desktops.map(desktop => (
        <DesktopLayer
          key={desktop.desktop.id}
          model={desktop}
          compact={presentation === "compact"}
          reducedMotion={reducedMotion}
          viewportReady={viewportState === "ready"}
          transition={transition}
          onTransitionComplete={onTransitionComplete}
          seamProjection={activeProjection}
          preview={preview}
          onResizeLayout={onResizeLayout}
        />
      ))}
      {viewportState === "rejected" ? (
        <div
          role="status"
          data-testid="os-viewport-rejected"
          className="absolute inset-0 grid place-items-center px-6"
        >
          <div className="max-w-sm border border-line bg-surface px-5 py-4 text-center shadow-overlay">
            <p className="text-body font-semibold text-foreground">
              This window is too narrow for the configured layout.
            </p>
            <p className="mt-1 text-small-body text-muted">
              Widen it or change the small viewport policy in Settings › Layouts.
            </p>
          </div>
        </div>
      ) : null}
    </div>
  );
}
