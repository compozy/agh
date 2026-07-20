// Suite: desktop menubar overlays
// Invariant: shell popovers and desktop dialogs have one active owner and unwind one layer per Esc.
// Boundary IN: DesktopMenubar, useDesktopOverlays, useOsShortcuts, and shared dialog/menu primitives.
// Boundary OUT: real browser focus integration and router journeys (web/e2e/__tests__/os-shell.spec.ts).
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { Dialog, DialogContent, DialogTitle } from "@agh/ui";

import type { WorkspacePayload } from "@/systems/workspace";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { useDesktopOverlays } from "../../hooks/use-desktop-overlays";
import { useOsShortcuts } from "../../hooks/use-os-shortcuts";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { DesktopMenubar } from "../desktop-menubar";

const WORKSPACES: WorkspacePayload[] = [
  {
    id: "w-agh",
    name: "agh",
    root_dir: "/work/agh",
    add_dirs: [],
    created_at: "2026-07-20T00:00:00Z",
    updated_at: "2026-07-20T00:00:00Z",
  },
  {
    id: "w-labs",
    name: "runtime-labs",
    root_dir: "/work/runtime-labs",
    add_dirs: [],
    created_at: "2026-07-20T00:00:00Z",
    updated_at: "2026-07-20T00:00:00Z",
  },
];

function createShell(): OsShellHandle {
  const store = createDesktopStore();
  const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
  store.getState().hydrate([]);
  return {
    store,
    coordinator: new RoutingCoordinator(store, port),
    flushPersistence: () => {},
  };
}

function MenubarOverlayHarness() {
  const overlays = useDesktopOverlays();

  useOsShortcuts({
    onPalette: () => overlays.toggleOverlay("palette"),
    onNewSession: () => {},
    onSpaces: () => overlays.toggleOverlay("spaces"),
    onEscape: () => {
      if (overlays.activeOverlay !== null) return;
      if (document.querySelector('[data-slot="dialog-content"]')) return;
      document.querySelector<HTMLElement>('[data-testid="shell-focus-target"]')?.focus();
    },
  });

  return (
    <div data-testid="shell-focus-target" tabIndex={-1}>
      <DesktopMenubar
        workspaces={WORKSPACES}
        activeWorkspace={WORKSPACES[0]}
        onSelectWorkspace={() => {}}
        onAddWorkspace={() => {}}
        onNewSession={() => {}}
        onOpenPalette={() => overlays.setOverlayOpen("palette", true)}
        onOpenSpaces={() => overlays.setOverlayOpen("spaces", true)}
        activeOverlay={overlays.activeOverlay}
        onOverlayOpenChange={overlays.setOverlayOpen}
      />

      <Dialog
        open={overlays.activeOverlay === "palette"}
        onOpenChange={open => overlays.setOverlayOpen("palette", open)}
      >
        <DialogContent>
          <DialogTitle>Command palette test</DialogTitle>
        </DialogContent>
      </Dialog>
      <Dialog
        open={overlays.activeOverlay === "spaces"}
        onOpenChange={open => overlays.setOverlayOpen("spaces", open)}
      >
        <DialogContent>
          <DialogTitle>Spaces overview test</DialogTitle>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function renderHarness() {
  render(
    <OsShellContext.Provider value={createShell()}>
      <MenubarOverlayHarness />
    </OsShellContext.Provider>
  );
}

describe("DesktopMenubar overlay coordination", () => {
  it("Should replace an open bell popover with the palette and return focus after two Esc presses", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Approvals" }));
    expect(screen.getByTestId("os-bell-popover")).toBeInTheDocument();

    fireEvent.keyDown(document.body, { key: "k", metaKey: true });
    expect(screen.getByRole("dialog", { name: "Command palette test" })).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByTestId("os-bell-popover")).toBeNull());

    fireEvent.keyDown(document.activeElement ?? document.body, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Command palette test" })).toBeNull()
    );

    fireEvent.keyDown(document.body, { key: "Escape" });
    expect(screen.getByTestId("shell-focus-target")).toHaveFocus();
  });

  it("Should expose Spaces through View and include its shortcut in Help", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.click(screen.getByRole("button", { name: "View" }));
    await user.click(await screen.findByRole("menuitem", { name: /Spaces overview/ }));
    expect(screen.getByRole("dialog", { name: "Spaces overview test" })).toBeInTheDocument();

    fireEvent.keyDown(document.activeElement ?? document.body, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Spaces overview test" })).toBeNull()
    );

    await user.click(screen.getByRole("button", { name: "Help" }));
    const shortcuts = await screen.findByTestId("os-help-shortcuts");
    expect(shortcuts).toHaveTextContent("Spaces overview");
    expect(shortcuts).toHaveTextContent("⇧⌘S");
  });
});
