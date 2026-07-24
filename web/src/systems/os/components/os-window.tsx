import { Suspense } from "react";
import { Rnd } from "react-rnd";

import { OverlayContainerContext, Spinner } from "@agh/ui";

import { cn } from "@/lib/utils";

import {
  OS_WINDOW_DRAG_CANCEL_SELECTOR,
  OS_WINDOW_DRAG_HANDLE_CLASS,
  useOsWindow,
} from "../hooks/use-os-window";
import { useDesktop } from "../hooks/use-desktop";
import { getOsApp, getOsAppMinimum } from "../lib/app-registry";
import type { OsWindow as OsWindowState } from "../lib/os-types";
import { OsWindowErrorBoundary } from "./os-window-error-boundary";
import { OsWindowFrame } from "./os-window-frame";
import { OsZoomMenu } from "./os-zoom-menu";

export interface OsWindowProps {
  windowId: string;
}

/** One daemon-managed window rendered from the current client projection. */
export function OsWindow({ windowId }: OsWindowProps) {
  const {
    win,
    focused,
    keepMounted,
    rect,
    rndRef,
    overlayHost,
    setOverlayHost,
    handleTrafficLight,
    handlePointerEnter,
    handlePointerDownCapture,
    handleFocusCapture,
    handleDragStart,
    handleDrag,
    handleDragStop,
    handleResizeStop,
  } = useOsWindow(windowId);
  const presentation = useDesktop(state => state.presentation);
  if (!win || !keepMounted) return null;

  const frame = (
    <WindowFrame
      focused={focused}
      onFocusCapture={handleFocusCapture}
      onOverlayHost={setOverlayHost}
      onPointerEnter={handlePointerEnter}
      onPointerDownCapture={handlePointerDownCapture}
      onTrafficLight={handleTrafficLight}
      overlayHost={overlayHost}
      presentation={presentation}
      win={win}
      windowId={windowId}
    />
  );

  // Compact (<960px): the same logical window as a full-bleed stack surface —
  // geometry untouched, z decides which one is visible (US-014.AC-1/AC-2).
  if (presentation === "compact") {
    return (
      <div
        className="absolute inset-0"
        style={{
          zIndex: win.layer,
          display: win.minimized || !win.stackActive ? "none" : undefined,
        }}
      >
        {frame}
      </div>
    );
  }

  if (rect === null) return null;
  const minimum = getOsAppMinimum(win.app);
  return (
    <Rnd
      ref={rndRef}
      position={{ x: rect.x, y: rect.y }}
      size={{ width: rect.w, height: rect.h }}
      minWidth={minimum.width}
      minHeight={minimum.height}
      disableDragging={!win.stackActive}
      enableResizing={win.placement === "floating"}
      resizeHandleClasses={{ bottomRight: "os-window-resize-handle" }}
      dragHandleClassName={OS_WINDOW_DRAG_HANDLE_CLASS}
      cancel={OS_WINDOW_DRAG_CANCEL_SELECTOR}
      onDragStart={handleDragStart}
      onDrag={handleDrag}
      onDragStop={handleDragStop}
      onResizeStop={handleResizeStop}
      style={{
        zIndex: win.layer,
        display: win.minimized || !win.stackActive ? "none" : undefined,
      }}
    >
      {frame}
    </Rnd>
  );
}

function WindowFrame({
  focused,
  onFocusCapture,
  onOverlayHost,
  onPointerEnter,
  onPointerDownCapture,
  onTrafficLight,
  overlayHost,
  presentation,
  win,
  windowId,
}: {
  focused: ReturnType<typeof useOsWindow>["focused"];
  onFocusCapture: ReturnType<typeof useOsWindow>["handleFocusCapture"];
  onOverlayHost: ReturnType<typeof useOsWindow>["setOverlayHost"];
  onPointerEnter: ReturnType<typeof useOsWindow>["handlePointerEnter"];
  onPointerDownCapture: ReturnType<typeof useOsWindow>["handlePointerDownCapture"];
  onTrafficLight: ReturnType<typeof useOsWindow>["handleTrafficLight"];
  overlayHost: ReturnType<typeof useOsWindow>["overlayHost"];
  presentation: "floating" | "compact";
  win: OsWindowState;
  windowId: string;
}) {
  const app = getOsApp(win.app);
  const Controller = app.Controller;
  const compact = presentation === "compact";

  return (
    <OsWindowFrame
      title={app.title}
      focused={focused}
      onTrafficLight={onTrafficLight}
      zoomMenu={
        compact ? undefined : button => <OsZoomMenu windowId={windowId}>{button}</OsZoomMenu>
      }
      presentation={presentation}
      headClassName={cn(
        !compact && `${OS_WINDOW_DRAG_HANDLE_CLASS} cursor-grab active:cursor-grabbing`
      )}
      className="relative h-full w-full"
      // Named region landmark per window: assistive tech can jump between
      // windows the way sighted users scan the desktop (WCAG landmarks).
      aria-label={`${app.title} window`}
      data-testid={`os-window-${windowId}`}
      data-app={win.app}
      data-minimized={win.minimized ? "" : undefined}
      data-window-placement={win.placement}
      onPointerEnter={onPointerEnter}
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
