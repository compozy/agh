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
import { getOsApp } from "../lib/app-registry";
import {
  OS_WINDOW_MIN_HEIGHT,
  OS_WINDOW_MIN_WIDTH,
  OS_WORK_AREA_INSETS,
  type OsWindow as OsWindowState,
} from "../lib/os-types";
import { OsWindowErrorBoundary } from "./os-window-error-boundary";
import { OsWindowFrame } from "./os-window-frame";
import { OsZoomMenu } from "./os-zoom-menu";

export interface OsWindowProps {
  windowId: string;
}

/**
 * One floating window: controlled `react-rnd` geometry under the WM store
 * (ADR-003 — transient gesture positions stay inside the drag mechanism; the
 * store commits at gesture end), the per-window overlay container (Modal &
 * Overlay Policy), and the minimize=unmount posture with the open-dialog
 * exemption (Safety Invariant 18). Maximized and snapped windows render the
 * same Rnd path with a locally derived rect (ADR-009, invariant 19) so they
 * stay resizable — dragging remains floating-only; the drag-away ghost moves
 * the same Rnd node so pointer capture and content identity survive every
 * state transition. Behavior lives in `useOsWindow`.
 */
export function OsWindow({ windowId }: OsWindowProps) {
  const {
    win,
    focused,
    keepMounted,
    derivedRect,
    ghostRect,
    minimizeTransform,
    rndRef,
    overlayHost,
    setOverlayHost,
    handleTrafficLight,
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
      minimizeTransform={minimizeTransform}
      onFocusCapture={handleFocusCapture}
      onOverlayHost={setOverlayHost}
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
        style={{ zIndex: win.z, display: win.minimized ? "none" : undefined }}
      >
        {frame}
      </div>
    );
  }

  // Pre-measure maximized fallback: before the win-layer reports bounds the
  // CSS insets fill the work area without any px math (never painted after
  // the first layout effect lands bounds).
  if (win.maximized && derivedRect === null) {
    const { top, right, bottom, left } = OS_WORK_AREA_INSETS;
    return (
      <div
        className="absolute"
        style={{
          left,
          top,
          right,
          bottom,
          zIndex: win.z,
          display: win.minimized ? "none" : undefined,
        }}
      >
        {frame}
      </div>
    );
  }

  // One Rnd across floating/snapped/maximized keeps the frame's element
  // identity stable — snap, unsnap, and zoom never remount window content,
  // and the drag-away ghost (ghostRect) moves the same node the pointer
  // capture lives on.
  const rect = ghostRect ?? derivedRect ?? win.rect;
  return (
    <Rnd
      ref={rndRef}
      position={{ x: rect.x, y: rect.y }}
      size={{ width: rect.w, height: rect.h }}
      minWidth={OS_WINDOW_MIN_WIDTH}
      minHeight={OS_WINDOW_MIN_HEIGHT}
      bounds="parent"
      disableDragging={win.maximized || win.snap !== null}
      resizeHandleClasses={{ bottomRight: "os-window-resize-handle" }}
      dragHandleClassName={OS_WINDOW_DRAG_HANDLE_CLASS}
      cancel={OS_WINDOW_DRAG_CANCEL_SELECTOR}
      onDragStart={handleDragStart}
      onDrag={handleDrag}
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
  minimizeTransform,
  onFocusCapture,
  onOverlayHost,
  onPointerDownCapture,
  onTrafficLight,
  overlayHost,
  presentation,
  win,
  windowId,
}: {
  focused: ReturnType<typeof useOsWindow>["focused"];
  minimizeTransform: ReturnType<typeof useOsWindow>["minimizeTransform"];
  onFocusCapture: ReturnType<typeof useOsWindow>["handleFocusCapture"];
  onOverlayHost: ReturnType<typeof useOsWindow>["setOverlayHost"];
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
        !compact &&
          !win.maximized &&
          `${OS_WINDOW_DRAG_HANDLE_CLASS} cursor-grab active:cursor-grabbing`
      )}
      className={cn("relative h-full w-full", minimizeTransform !== null && "os-window-minimizing")}
      style={minimizeTransform !== null ? { transform: minimizeTransform, opacity: 0 } : undefined}
      // Named region landmark per window: assistive tech can jump between
      // windows the way sighted users scan the desktop (WCAG landmarks).
      aria-label={`${app.title} window`}
      data-testid={`os-window-${windowId}`}
      data-app={win.app}
      data-minimized={win.minimized ? "" : undefined}
      data-snapped={win.snap !== null ? "" : undefined}
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
