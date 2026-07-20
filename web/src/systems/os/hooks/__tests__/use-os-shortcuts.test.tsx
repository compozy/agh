// Suite: global OS shortcuts
// Invariant: each documented shortcut dispatches exactly to its shell/window owner.
// Boundary IN: document keyboard events, desktop store, and routing coordinator actions.
// Boundary OUT: browser-reserved keys and visible overlay rendering (component/E2E suites).
import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { useOsShortcuts, type OsShortcutHandlers } from "../use-os-shortcuts";

function Harness({ handlers }: { handlers: OsShortcutHandlers }) {
  useOsShortcuts(handlers);
  return <input aria-label="Anywhere" data-testid="anywhere-input" />;
}

function createHarness() {
  const store = createDesktopStore();
  const pushes: string[] = [];
  const port: OsRouterPort = {
    navigate: location => pushes.push(location.pathname),
    replace: location => pushes.push(location.pathname),
  };
  const coordinator = new RoutingCoordinator(store, port);
  store.getState().hydrate([]);
  coordinator.completeHydration();
  const shell: OsShellHandle = { store, coordinator, flushPersistence: () => {} };
  return { store, coordinator, shell, pushes };
}

describe("useOsShortcuts", () => {
  it("Should open the palette on ⌘K from anywhere — including inputs — and ignore ⌘J (UT-060)", () => {
    const { shell } = createHarness();
    const onPalette = vi.fn();
    const onNewSession = vi.fn();
    const onSpaces = vi.fn();
    const onEscape = vi.fn();
    const { getByTestId } = render(
      <OsShellContext.Provider value={shell}>
        <Harness handlers={{ onPalette, onNewSession, onSpaces, onEscape }} />
      </OsShellContext.Provider>
    );

    fireEvent.keyDown(document.body, { key: "k", metaKey: true });
    expect(onPalette).toHaveBeenCalledTimes(1);

    // Global takeover: ⌘K fires even while an input owns focus.
    const input = getByTestId("anywhere-input");
    input.focus();
    fireEvent.keyDown(input, { key: "k", metaKey: true });
    expect(onPalette).toHaveBeenCalledTimes(2);

    // ⌘J belongs to the RuntimeSelector's composer scope, never the palette
    // (its composer-scoped behavior is covered by the runtime-selector suite).
    fireEvent.keyDown(document.body, { key: "j", metaKey: true });
    expect(onPalette).toHaveBeenCalledTimes(2);
    expect(onNewSession).not.toHaveBeenCalled();
    expect(onSpaces).not.toHaveBeenCalled();
    expect(onEscape).not.toHaveBeenCalled();
  });

  it("Should route ⌘N/⇧⌘S/Esc and ⌘W/⌘M to their shell owners", () => {
    const { store, coordinator, shell, pushes } = createHarness();
    coordinator.userOpen({ app: "tasks" });
    pushes.length = 0;
    const onNewSession = vi.fn();
    const onSpaces = vi.fn();
    const onEscape = vi.fn();
    render(
      <OsShellContext.Provider value={shell}>
        <Harness handlers={{ onPalette: vi.fn(), onNewSession, onSpaces, onEscape }} />
      </OsShellContext.Provider>
    );

    fireEvent.keyDown(document.body, { key: "n", metaKey: true });
    expect(onNewSession).toHaveBeenCalledTimes(1);

    fireEvent.keyDown(document.body, { key: "s", metaKey: true, shiftKey: true });
    expect(onSpaces).toHaveBeenCalledTimes(1);

    fireEvent.keyDown(document.body, { key: "s", metaKey: true });
    expect(onSpaces).toHaveBeenCalledTimes(1);

    fireEvent.keyDown(document.body, { key: "Escape" });
    expect(onEscape).toHaveBeenCalledTimes(1);

    fireEvent.keyDown(document.body, { key: "m", metaKey: true });
    expect(store.getState().windows["app:tasks"].minimized).toBe(true);

    store.getState().restoreWindow("app:tasks");
    fireEvent.keyDown(document.body, { key: "w", metaKey: true });
    expect(store.getState().windows["app:tasks"]).toBeUndefined();
  });
});
