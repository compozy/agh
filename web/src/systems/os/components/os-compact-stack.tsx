import { useShallow } from "zustand/shallow";

import { Kbd } from "@agh/ui";

import { useDesktop } from "../hooks/use-desktop";
import { OsWindow } from "./os-window";

/**
 * Compact (<960px) presentation (os-v2.css mobile block): the same logical
 * windows render as full-bleed stacked surfaces — z decides which one is
 * visible, geometry is untouched so returning to floating restores every rect
 * (US-014.AC-2). The stack sits above the tab-bar dock's safe-area strip;
 * window-scoped dialogs are naturally full-viewport here (US-020.EC-4).
 */
export function OsCompactStack() {
  const windowIds = useDesktop(useShallow(state => Object.keys(state.windows).sort()));
  const anyVisible = useDesktop(state => Object.values(state.windows).some(win => !win.minimized));

  return (
    <div
      data-slot="os-compact-stack"
      className="absolute inset-x-0 top-0 bottom-[calc(var(--height-dock-tabbar)+env(safe-area-inset-bottom,0px))] overflow-hidden"
    >
      {!anyVisible ? (
        <p
          data-testid="os-desk-hint"
          className="pointer-events-none absolute top-1/2 left-1/2 flex -translate-x-1/2 -translate-y-1/2 items-center gap-2 px-6 text-center text-small-body text-subtle select-none"
        >
          <Kbd>⌘K</Kbd> to open anything
        </p>
      ) : null}
      {windowIds.map(id => (
        <OsWindow key={id} windowId={id} />
      ))}
    </div>
  );
}
