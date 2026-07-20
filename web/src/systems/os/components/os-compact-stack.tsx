import { Suspense } from "react";
import { useShallow } from "zustand/shallow";

import { Kbd, Spinner } from "@agh/ui";

import { useDesktop } from "../hooks/use-desktop";
import { getOsApp } from "../lib/app-registry";
import { OsWindowErrorBoundary } from "./os-window-error-boundary";

/**
 * Compact (<960px) presentation: the focused window fills the viewport as a
 * full-bleed surface; geometry is untouched so returning to floating restores
 * every rect (US-014). The full compact visual pass (tab-bar dock, safe-area
 * insets) lands with the compact-mode task; this keeps the same logical
 * windows usable below the breakpoint.
 */
export function OsCompactStack() {
  const focused = useDesktop(
    useShallow(state => {
      const win = state.focusedId !== null ? state.windows[state.focusedId] : null;
      return win ? { id: win.id, app: win.app } : null;
    })
  );

  if (!focused) {
    return (
      <div className="absolute inset-0 grid place-items-center">
        <p
          data-testid="os-desk-hint"
          className="flex items-center gap-2 text-small-body text-subtle select-none"
        >
          <Kbd>⌘K</Kbd> to open anything
        </p>
      </div>
    );
  }

  const app = getOsApp(focused.app);
  const Controller = app.Controller;

  return (
    <div
      data-slot="os-compact-stack"
      data-testid={`os-window-${focused.id}`}
      data-app={focused.app}
      className="absolute inset-0 flex min-h-0 flex-col overflow-hidden bg-canvas"
      aria-label={app.title}
    >
      <OsWindowErrorBoundary title={app.title}>
        <Suspense
          fallback={
            <div className="flex flex-1 items-center justify-center">
              <Spinner className="size-4 text-subtle" />
            </div>
          }
        >
          <Controller windowId={focused.id} />
        </Suspense>
      </OsWindowErrorBoundary>
    </div>
  );
}
