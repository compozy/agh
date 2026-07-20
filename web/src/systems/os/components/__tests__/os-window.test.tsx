import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

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
        <OsWindow key={id} windowId={id} rootCrumb="agh" />
      ))}
    </OsShellContext.Provider>
  );
}

describe("OsWindow", () => {
  it("Should focus an unfocused window on keyboard activation exactly like pointer (UT-081, rule 5)", async () => {
    const { store, coordinator, shell, pushes } = createHarness();
    coordinator.userOpen({ app: "tasks" });
    coordinator.userOpen({ app: "vault" });
    pushes.length = 0;

    renderWindows(shell, ["app:tasks", "app:vault"]);
    await screen.findAllByTestId("os-pending-app");

    // Keyboard: focus lands on a control inside the unfocused tasks window.
    const tasksWindow = screen.getByTestId("os-window-app:tasks");
    const closeButton = tasksWindow.querySelector<HTMLButtonElement>('[data-action="close"]');
    expect(closeButton).not.toBeNull();
    fireEvent.focus(closeButton as HTMLButtonElement);

    expect(store.getState().focusedId).toBe("app:tasks");
    expect(pushes).toEqual(["/tasks"]);

    // Pointer parity: pointerdown into the (now unfocused) vault window.
    const vaultWindow = screen.getByTestId("os-window-app:vault");
    fireEvent.pointerDown(vaultWindow);
    expect(store.getState().focusedId).toBe("app:vault");
    expect(pushes).toEqual(["/tasks", "/vault"]);
  });

  it("Should keep a minimized window mounted while its dialog is open and unmount after it closes (UT-086, invariant 18)", async () => {
    const { store, coordinator, shell } = createHarness();
    coordinator.userOpen({ app: "tasks" });

    renderWindows(shell, ["app:tasks"]);
    await screen.findByTestId("os-pending-app");

    // A window-scoped dialog portals into the window's overlay host — the
    // same seam `OverlayContainerContext` gives the @agh/ui Dialog.
    const host = screen
      .getByTestId("os-window-app:tasks")
      .querySelector('[data-slot="os-window-overlays"]') as HTMLElement;
    expect(host).not.toBeNull();
    const dialogNode = document.createElement("div");
    dialogNode.setAttribute("role", "dialog");
    dialogNode.textContent = "unsaved form";
    host.appendChild(dialogNode);
    await waitFor(() => expect(host.childElementCount).toBe(1));

    store.getState().minimizeWindow("app:tasks");
    // Hidden but mounted: the frame stays in the DOM with its dialog intact.
    await waitFor(() => {
      const frame = screen.getByTestId("os-window-app:tasks");
      expect(frame).toHaveAttribute("data-minimized");
    });
    expect(screen.getByText("unsaved form")).toBeInTheDocument();

    // Restore before close: the dialog is exactly as left.
    store.getState().restoreWindow("app:tasks");
    await waitFor(() =>
      expect(screen.getByTestId("os-window-app:tasks")).not.toHaveAttribute("data-minimized")
    );
    expect(screen.getByText("unsaved form")).toBeInTheDocument();

    // Minimize again, then close the dialog: the unmount completes.
    store.getState().minimizeWindow("app:tasks");
    dialogNode.remove();
    await waitFor(() => expect(screen.queryByTestId("os-window-app:tasks")).toBeNull());
  });

  it("Should unmount the body outright when minimized without an open dialog (minimize=unmount posture)", async () => {
    const { store, coordinator, shell } = createHarness();
    coordinator.userOpen({ app: "tasks" });
    renderWindows(shell, ["app:tasks"]);
    await screen.findByTestId("os-pending-app");

    store.getState().minimizeWindow("app:tasks");
    await waitFor(() => expect(screen.queryByTestId("os-window-app:tasks")).toBeNull());

    // The WM entry survives with its rect (only the body unmounted).
    expect(store.getState().windows["app:tasks"].minimized).toBe(true);
  });
});
