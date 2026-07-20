import { Suspense } from "react";
import { Rnd } from "react-rnd";

import { OverlayContainerContext, Spinner } from "@agh/ui";

import { cn } from "@/lib/utils";

import { useOsWindow, type OsWindowModel } from "../hooks/use-os-window";
import { getOsApp } from "../lib/app-registry";
import type { OsWindow as OsWindowState } from "../lib/os-types";
import { OsWindowErrorBoundary } from "./os-window-error-boundary";
import { OsWindowFrame } from "./os-window-frame";

const MIN_WINDOW_WIDTH = 280;
const MIN_WINDOW_HEIGHT = 180;
const DRAG_HANDLE_CLASS = "os-window-drag-handle";
/** Head controls that must never start a drag. */
const DRAG_CANCEL_SELECTOR = '[data-slot="os-traffic-lights"], [data-slot="topbar-trailing"]';

export interface OsWindowProps {
  windowId: string;
  /** Workspace crumb rendered before the window title (`<ws> / <title>`). */
  rootCrumb: string;
}

/**
 * One floating window: controlled `react-rnd` geometry under the WM store
 * (ADR-003 — transient gesture positions stay inside the drag mechanism; the
 * store commits at gesture end), the per-window overlay container (Modal &
 * Overlay Policy), and the minimize=unmount posture with the open-dialog
 * exemption (Safety Invariant 18). Behavior lives in `useOsWindow`.
 */
export function OsWindow({ windowId, rootCrumb }: OsWindowProps) {
  const model = useOsWindow(windowId);
  const { win, keepMounted } = model;
  if (!win || !keepMounted) return null;

  const frame = <WindowFrame model={model} win={win} windowId={windowId} rootCrumb={rootCrumb} />;

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
      dragHandleClassName={DRAG_HANDLE_CLASS}
      cancel={DRAG_CANCEL_SELECTOR}
      onDragStop={model.handleDragStop}
      onResizeStop={model.handleResizeStop}
      style={{ zIndex: win.z, display: win.minimized ? "none" : undefined }}
    >
      {frame}
    </Rnd>
  );
}

function WindowFrame({
  model,
  win,
  windowId,
  rootCrumb,
}: {
  model: OsWindowModel;
  win: OsWindowState;
  windowId: string;
  rootCrumb: string;
}) {
  const app = getOsApp(win.app);
  const Controller = app.Controller;

  return (
    <OsWindowFrame
      title={app.title}
      rootCrumb={rootCrumb}
      focused={model.focused}
      onTrafficLight={model.handleTrafficLight}
      headClassName={cn(
        !win.maximized && `${DRAG_HANDLE_CLASS} cursor-grab active:cursor-grabbing`
      )}
      className="relative h-full w-full"
      data-testid={`os-window-${windowId}`}
      data-app={win.app}
      data-minimized={win.minimized ? "" : undefined}
      onPointerDownCapture={model.handlePointerDownCapture}
      onFocusCapture={model.handleFocusCapture}
    >
      <OverlayContainerContext.Provider value={model.overlayHost}>
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
      </OverlayContainerContext.Provider>
      <div
        ref={model.setOverlayHost}
        data-slot="os-window-overlays"
        // contain-paint makes portaled fixed-position scrims resolve against
        // this box, so a dialog dims and travels with its window only.
        className="contain-paint pointer-events-none absolute inset-0 z-40 [&:not(:empty)]:pointer-events-auto"
      />
    </OsWindowFrame>
  );
}
