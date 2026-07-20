import { Suspense } from "react";
import { Rnd } from "react-rnd";

import { OverlayContainerContext, Spinner } from "@agh/ui";

import { cn } from "@/lib/utils";

import { useOsWindow } from "../hooks/use-os-window";
import { getOsApp } from "../lib/app-registry";
import type { OsWindow as OsWindowState } from "../lib/os-types";
import { OsWindowErrorBoundary } from "./os-window-error-boundary";
import { OsWindowFrame } from "./os-window-frame";

const MIN_WINDOW_WIDTH = 280;
const MIN_WINDOW_HEIGHT = 180;
const DRAG_HANDLE_CLASS = "os-window-drag-handle";
/** Head controls that must never start a drag. */
const DRAG_CANCEL_SELECTOR = [
  '[data-slot="os-traffic-lights"]',
  '[data-slot="topbar-back"]',
  '[data-slot="topbar-crumb"]',
  '[data-slot="topbar-crumb-more"]',
  '[data-slot="topbar-route-nav"]',
  '[data-slot="topbar-trailing"]',
].join(", ");

export interface OsWindowProps {
  windowId: string;
}

/**
 * One floating window: controlled `react-rnd` geometry under the WM store
 * (ADR-003 — transient gesture positions stay inside the drag mechanism; the
 * store commits at gesture end), the per-window overlay container (Modal &
 * Overlay Policy), and the minimize=unmount posture with the open-dialog
 * exemption (Safety Invariant 18). Behavior lives in `useOsWindow`.
 */
export function OsWindow({ windowId }: OsWindowProps) {
  const {
    win,
    focused,
    keepMounted,
    overlayHost,
    setOverlayHost,
    handleTrafficLight,
    handlePointerDownCapture,
    handleFocusCapture,
    handleDragStop,
    handleResizeStop,
  } = useOsWindow(windowId);
  if (!win || !keepMounted) return null;

  const frame = (
    <WindowFrame
      focused={focused}
      onFocusCapture={handleFocusCapture}
      onOverlayHost={setOverlayHost}
      onPointerDownCapture={handlePointerDownCapture}
      onTrafficLight={handleTrafficLight}
      overlayHost={overlayHost}
      win={win}
      windowId={windowId}
    />
  );

  if (win.maximized) {
    return (
      <div
        className="absolute"
        style={{
          left: 10,
          top: 8,
          right: 10,
          bottom: 78,
          zIndex: win.z,
          display: win.minimized ? "none" : undefined,
        }}
      >
        {frame}
      </div>
    );
  }

  return (
    <Rnd
      position={{ x: win.rect.x, y: win.rect.y }}
      size={{ width: win.rect.w, height: win.rect.h }}
      minWidth={MIN_WINDOW_WIDTH}
      minHeight={MIN_WINDOW_HEIGHT}
      bounds="parent"
      resizeHandleClasses={{ bottomRight: "os-window-resize-handle" }}
      dragHandleClassName={DRAG_HANDLE_CLASS}
      cancel={DRAG_CANCEL_SELECTOR}
      onDragStop={handleDragStop}
      onResizeStop={handleResizeStop}
      style={{ zIndex: win.z, display: win.minimized ? "none" : undefined }}
    >
      {frame}
    </Rnd>
  );
}

function WindowFrame({
  focused,
  onFocusCapture,
  onOverlayHost,
  onPointerDownCapture,
  onTrafficLight,
  overlayHost,
  win,
  windowId,
}: {
  focused: ReturnType<typeof useOsWindow>["focused"];
  onFocusCapture: ReturnType<typeof useOsWindow>["handleFocusCapture"];
  onOverlayHost: ReturnType<typeof useOsWindow>["setOverlayHost"];
  onPointerDownCapture: ReturnType<typeof useOsWindow>["handlePointerDownCapture"];
  onTrafficLight: ReturnType<typeof useOsWindow>["handleTrafficLight"];
  overlayHost: ReturnType<typeof useOsWindow>["overlayHost"];
  win: OsWindowState;
  windowId: string;
}) {
  const app = getOsApp(win.app);
  const Controller = app.Controller;

  return (
    <OsWindowFrame
      title={app.title}
      focused={focused}
      onTrafficLight={onTrafficLight}
      headClassName={cn(
        !win.maximized && `${DRAG_HANDLE_CLASS} cursor-grab active:cursor-grabbing`
      )}
      className="relative h-full w-full"
      data-testid={`os-window-${windowId}`}
      data-app={win.app}
      data-minimized={win.minimized ? "" : undefined}
      onPointerDownCapture={onPointerDownCapture}
      onFocusCapture={onFocusCapture}
    >
      <OverlayContainerContext.Provider value={overlayHost}>
        {overlayHost ? (
          <OsWindowErrorBoundary title={app.title}>
            <Suspense
              fallback={
                <div className="flex min-h-32 flex-1 items-center justify-center">
                  <Spinner className="size-4 text-subtle" />
                </div>
              }
            >
              <Controller windowId={windowId} />
            </Suspense>
          </OsWindowErrorBoundary>
        ) : (
          <div className="flex min-h-32 flex-1 items-center justify-center">
            <Spinner className="size-4 text-subtle" />
          </div>
        )}
      </OverlayContainerContext.Provider>
      <div
        ref={onOverlayHost}
        data-slot="os-window-overlays"
        // contain-paint makes portaled fixed-position scrims resolve against
        // this box, so a dialog dims and travels with its window only.
        className="contain-paint pointer-events-none absolute inset-0 z-40 [&:not(:empty)]:pointer-events-auto"
      />
    </OsWindowFrame>
  );
}
