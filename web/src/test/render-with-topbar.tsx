import { render as rtlRender, type RenderResult } from "@testing-library/react";
import type { ReactElement } from "react";

import { Topbar, TopbarSlotProvider, type TopbarRouteContext } from "@agh/ui";

interface RenderWithTopbarResult extends RenderResult {
  rerender: (ui: ReactElement) => void;
}

/**
 * Test helper that mounts a route component under a TopbarSlotProvider plus a
 * stub `<Topbar>`, so any `useTopbarSlot` calls actually render their slot
 * content (search/tabs/actions) into the test DOM.
 *
 * The returned `rerender` re-applies the wrapper so callers that mutate state
 * between renders still get the slot context.
 */
export function renderWithTopbar(
  ui: ReactElement,
  routeContext: TopbarRouteContext = { title: "Test" }
): RenderWithTopbarResult {
  const wrap = (child: ReactElement) => (
    <TopbarSlotProvider>
      <Topbar route={routeContext} />
      {child}
    </TopbarSlotProvider>
  );
  const result = rtlRender(wrap(ui));
  return {
    ...result,
    rerender: (next: ReactElement) => result.rerender(wrap(next)),
  };
}
