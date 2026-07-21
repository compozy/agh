import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("react-rnd", () => ({
  Rnd: ({ children, cancel }: { children: ReactNode; cancel?: string }) => (
    <div data-testid="rnd-window" data-drag-cancel={cancel}>
      {children}
    </div>
  ),
}));

vi.mock("../../lib/app-registry", async importOriginal => {
  const actual = await importOriginal<typeof import("../../lib/app-registry")>();
  return {
    ...actual,
    getOsApp: (id: Parameters<typeof actual.getOsApp>[0]) => ({
      ...actual.getOsApp(id),
      Controller: () => <div data-testid="os-pending-app" />,
    }),
  };
});

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { OsWindow } from "../os-window";

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

function renderWindows(shell: OsShellHandle, ids: string[]) {
  return render(
    <OsShellContext.Provider value={shell}>
      {ids.map(id => (
        <OsWindow key={id} windowId={id} />
      ))}
    </OsShellContext.Provider>
  );
}

describe("OsWindow", () => {
  it("Should focus an unfocused window on keyboard activation exactly like pointer (UT-081, rule 5)", async () => {
    const { store, coordinator, shell, pushes } = createHarness();
    coordinator.userOpen({ app: "sandbox" });
    coordinator.userOpen({ app: "vault" });
    pushes.length = 0;

    renderWindows(shell, ["app:sandbox", "app:vault"]);
    await screen.findAllByTestId("os-pending-app");

    // Keyboard: focus lands on a control inside the unfocused sandbox window.
    const sandboxWindow = screen.getByTestId("os-window-app:sandbox");
    const closeButton = sandboxWindow.querySelector<HTMLButtonElement>('[data-action="close"]');
    expect(closeButton).not.toBeNull();
    fireEvent.focus(closeButton as HTMLButtonElement);

    expect(store.getState().focusedId).toBe("app:sandbox");
    expect(pushes).toEqual(["/sandbox"]);

    // Pointer parity: pointerdown into the (now unfocused) vault window.
    const vaultWindow = screen.getByTestId("os-window-app:vault");
    fireEvent.pointerDown(vaultWindow);
    expect(store.getState().focusedId).toBe("app:vault");
    expect(pushes).toEqual(["/sandbox", "/vault"]);
  });

  it("Should keep a minimized window mounted while its dialog is open and unmount after it closes (UT-086, invariant 18)", async () => {
    const { store, coordinator, shell } = createHarness();
    coordinator.userOpen({ app: "vault" });

    renderWindows(shell, ["app:vault"]);
    await screen.findByTestId("os-pending-app");

    // A window-scoped dialog portals into the window's overlay host — the
    // same seam `OverlayContainerContext` gives the @agh/ui Dialog.
    const host = screen
      .getByTestId("os-window-app:vault")
      .querySelector('[data-slot="os-window-overlays"]') as HTMLElement;
    expect(host).not.toBeNull();
    const dialogNode = document.createElement("div");
    dialogNode.setAttribute("role", "dialog");
    dialogNode.textContent = "unsaved form";
    await act(async () => {
      host.appendChild(dialogNode);
    });
    await waitFor(() => expect(host.childElementCount).toBe(1));

    act(() => store.getState().minimizeWindow("app:vault"));
    // Hidden but mounted: the frame stays in the DOM with its dialog intact.
    await waitFor(() => {
      const frame = screen.getByTestId("os-window-app:vault");
      expect(frame).toHaveAttribute("data-minimized");
    });
    expect(screen.getByText("unsaved form")).toBeInTheDocument();

    // Restore before close: the dialog is exactly as left.
    act(() => store.getState().restoreWindow("app:vault"));
    await waitFor(() =>
      expect(screen.getByTestId("os-window-app:vault")).not.toHaveAttribute("data-minimized")
    );
    expect(screen.getByText("unsaved form")).toBeInTheDocument();

    // Minimize again, then close the dialog: the unmount completes.
    act(() => store.getState().minimizeWindow("app:vault"));
    await act(async () => {
      dialogNode.remove();
    });
    await waitFor(() => expect(screen.queryByTestId("os-window-app:vault")).toBeNull());
  });

  it("Should unmount the body outright when minimized without an open dialog (minimize=unmount posture)", async () => {
    const { store, coordinator, shell } = createHarness();
    coordinator.userOpen({ app: "vault" });
    renderWindows(shell, ["app:vault"]);
    await screen.findByTestId("os-pending-app");

    act(() => store.getState().minimizeWindow("app:vault"));
    await waitFor(() => expect(screen.queryByTestId("os-window-app:vault")).toBeNull());

    // The WM entry survives with its rect (only the body unmounted).
    expect(store.getState().windows["app:vault"].minimized).toBe(true);
  });

  it("Should exclude every interactive head navigation zone from the drag handle", async () => {
    const { coordinator, shell } = createHarness();
    coordinator.userOpen({ app: "tasks" });

    renderWindows(shell, ["app:tasks"]);
    await screen.findByTestId("os-pending-app");

    const dragCancel = screen.getByTestId("rnd-window").getAttribute("data-drag-cancel") ?? "";
    expect(dragCancel).toContain('[data-slot="topbar-back"]');
    expect(dragCancel).toContain('[data-slot="topbar-crumb"]');
    expect(dragCancel).toContain('[data-slot="topbar-crumb-more"]');
    expect(dragCancel).toContain('[data-slot="topbar-nav"]');
  });

  it("Should present compact windows as full-bleed stack surfaces without Rnd, drag, or zoom (US-014.AC-1)", async () => {
    const { store, coordinator, shell } = createHarness();
    coordinator.userOpen({ app: "tasks" });
    act(() => store.getState().setPresentation("compact"));

    renderWindows(shell, ["app:tasks"]);
    await screen.findByTestId("os-pending-app");

    // No Rnd wrapper — geometry is CSS-forced, never gesture-driven.
    expect(screen.queryByTestId("rnd-window")).toBeNull();
    const frame = screen.getByTestId("os-window-app:tasks");
    expect(frame).toHaveAttribute("data-presentation", "compact");
    // The zoom control disappears; close/minimize keep their labels.
    expect(screen.queryByRole("button", { name: "Zoom window" })).toBeNull();
    expect(screen.getByRole("button", { name: "Close window" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Minimize window" })).toBeInTheDocument();
    // The head is not a drag handle in the stack.
    expect(frame.querySelector(".os-window-drag-handle")).toBeNull();
  });
});
