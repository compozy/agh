// Suite: OS dock state
// Invariant: every launcher exposes closed, running, or minimized from its real item state.
// Boundary IN: OsDock item semantics.
// Boundary OUT: WM-to-dock projection (DesktopDock) and browser lifecycle journeys.
import { render, screen } from "@testing-library/react";
import { Bot, LayoutDashboard, ListChecks } from "lucide-react";
import { describe, expect, it, vi } from "vitest";

import { OsDock } from "../os-dock";

describe("OsDock", () => {
  it("Should expose the real closed, running, and minimized state for each launcher", () => {
    render(
      <OsDock
        items={[
          { id: "dashboard", name: "Dashboard", icon: LayoutDashboard, running: true },
          { id: "tasks", name: "Tasks", icon: ListChecks, minimized: true },
          { id: "agents", name: "Agents", icon: Bot },
        ]}
        onSelect={vi.fn()}
      />
    );

    expect(screen.getByRole("button", { name: "Dashboard" })).toHaveAttribute(
      "data-state",
      "running"
    );
    expect(screen.getByRole("button", { name: "Tasks" })).toHaveAttribute(
      "data-state",
      "minimized"
    );
    expect(screen.getByRole("button", { name: "Agents" })).toHaveAttribute("data-state", "closed");
  });

  it("Should hide zero badges and cap large attention counts at 9+ (UT-066)", () => {
    render(
      <OsDock
        items={[
          { id: "dashboard", name: "Dashboard", icon: LayoutDashboard, badge: 0 },
          { id: "tasks", name: "Tasks", icon: ListChecks, badge: 12 },
        ]}
        onSelect={vi.fn()}
      />
    );

    expect(screen.getByRole("button", { name: "Dashboard" })).not.toHaveTextContent("0");
    expect(screen.getByRole("button", { name: "Tasks" })).toHaveTextContent("9+");
  });
});
