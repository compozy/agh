// Suite: OS sessions rail
// Invariant: the rail filters catalog truth, persists group state, and restores compact focus.
// Boundary IN: DesktopSessionsRail, desktop store, and routing coordinator.
// Boundary OUT: session catalog transport and full browser window journeys.
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import type { SessionPayload } from "@/systems/session";

import { OsShellContext, type OsShellHandle } from "../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../lib/routing-coordinator";
import { createDesktopStore } from "../../stores/desktop-store";
import { DesktopSessionsRail } from "../sessions-rail";

function session(overrides: Partial<SessionPayload> = {}): SessionPayload {
  return {
    id: "session-1",
    name: "Web shell polish",
    agent_name: "codex",
    provider: "codex",
    workspace_id: "workspace-1",
    workspace_path: "/workspace/agh",
    state: "active",
    badge: "running",
    attachable: true,
    available_commands: [],
    created_at: "2026-07-20T12:00:00Z",
    updated_at: "2026-07-20T12:01:00Z",
    ...overrides,
  };
}

const SESSIONS: SessionPayload[] = [
  session(),
  session({ id: "session-2", name: "Runtime audit", agent_name: "webgen", badge: "idle" }),
  session({ id: "session-3", name: "Release notes", agent_name: "codex", badge: "stopped" }),
];

function createShell(): OsShellHandle {
  const store = createDesktopStore();
  const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
  store.getState().hydrate([]);
  const coordinator = new RoutingCoordinator(store, port);
  coordinator.completeHydration();
  return { store, coordinator, flushPersistence: () => {} };
}

function renderRail(shell: OsShellHandle) {
  return render(
    <OsShellContext.Provider value={shell}>
      <DesktopSessionsRail sessions={SESSIONS} disconnected={false} />
    </OsShellContext.Provider>
  );
}

describe("DesktopSessionsRail", () => {
  it("Should filter live by title or agent and restore the full catalog when cleared (UT-067)", async () => {
    const user = userEvent.setup();
    const shell = createShell();
    shell.store.getState().openRail();
    renderRail(shell);

    const filter = screen.getByRole("searchbox", { name: "Filter sessions" });
    await user.type(filter, "web");
    expect(screen.getAllByTestId("os-rail-session-session-1")).not.toHaveLength(0);
    expect(screen.getAllByTestId("os-rail-session-session-2")).not.toHaveLength(0);
    expect(screen.queryByTestId("os-rail-session-session-3")).toBeNull();

    await user.clear(filter);
    expect(screen.getAllByTestId("os-rail-session-session-3")).not.toHaveLength(0);
  });

  it("Should retain an agent collapse after the rail remounts (UT-068)", async () => {
    const user = userEvent.setup();
    const shell = createShell();
    shell.store.getState().openRail();
    const first = renderRail(shell);

    await user.click(screen.getByRole("button", { name: "Show all sessions" }));
    const group = screen.getByRole("button", { name: /codex/i, expanded: true });
    await user.click(group);
    expect(group).toHaveAttribute("aria-expanded", "false");
    expect(shell.store.getState().railCollapsedAgentIds).toEqual(["codex"]);

    first.unmount();
    renderRail(shell);
    await user.click(screen.getByRole("button", { name: "Show all sessions" }));
    expect(screen.getByRole("button", { name: /codex/i, expanded: false })).toHaveAttribute(
      "aria-expanded",
      "false"
    );
  });

  it("Should render compact as a sheet and restore focus on dismissal (UT-084)", async () => {
    const user = userEvent.setup();
    const shell = createShell();
    shell.store.getState().setPresentation("compact");

    render(
      <OsShellContext.Provider value={shell}>
        <button type="button" onClick={() => shell.store.getState().openRail()}>
          Open sessions
        </button>
        <DesktopSessionsRail sessions={SESSIONS} disconnected={false} />
      </OsShellContext.Provider>
    );

    const trigger = screen.getByRole("button", { name: "Open sessions" });
    await user.click(trigger);
    expect(screen.getByTestId("os-sessions-rail-sheet")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close sessions" }));

    await waitFor(() => expect(screen.queryByTestId("os-sessions-rail-sheet")).toBeNull());
    expect(trigger).toHaveFocus();
  });
});
