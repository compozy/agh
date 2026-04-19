import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { UIProvider } from "@agh/ui";
import { describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "../types";
import { AgentSidebarGroup } from "./agent-sidebar-group";

const mockAgent: AgentPayload = {
  name: "claude-agent",
  provider: "claude",
  prompt: "You are a helpful assistant",
};

function renderGroup(
  props: Partial<React.ComponentProps<typeof AgentSidebarGroup>> = {},
  children?: React.ReactNode
) {
  return render(
    <UIProvider reducedMotion="always">
      <AgentSidebarGroup agent={mockAgent} {...props}>
        {children}
      </AgentSidebarGroup>
    </UIProvider>
  );
}

describe("AgentSidebarGroup", () => {
  it("renders agent name + provider glyph inside the trigger", () => {
    renderGroup();
    expect(screen.getByText("claude-agent")).toBeInTheDocument();
    expect(screen.getByTestId("agent-sidebar-group-trigger-claude-agent")).toBeInTheDocument();
    expect(
      screen
        .getByTestId("agent-sidebar-group-trigger-claude-agent")
        .querySelector('[data-slot="agent-icon"]')
    ).not.toBeNull();
  });

  it("shows a New Session action button", () => {
    renderGroup();
    expect(screen.getByRole("button", { name: "New Session" })).toBeInTheDocument();
  });

  it("calls onNewSession with agent name when clicking New Session", async () => {
    const user = userEvent.setup();
    const onNewSession = vi.fn();
    renderGroup({ onNewSession });

    await user.click(screen.getByTestId("agent-sidebar-group-new-session-claude-agent"));

    expect(onNewSession).toHaveBeenCalledWith("claude-agent");
  });

  it("disables the new-session action when newSessionDisabled is true", async () => {
    const user = userEvent.setup();
    const onNewSession = vi.fn();
    renderGroup({ onNewSession, newSessionDisabled: true });

    const action = screen.getByRole("button", { name: "New Session" });
    expect(action).toBeDisabled();
    await user.click(action);
    expect(onNewSession).not.toHaveBeenCalled();
  });

  it("renders one group per agent", () => {
    const agents: AgentPayload[] = [
      { name: "agent-1", provider: "claude", prompt: "prompt1" },
      { name: "agent-2", provider: "codex", prompt: "prompt2" },
      { name: "agent-3", provider: "gemini", prompt: "prompt3" },
    ];

    render(
      <UIProvider reducedMotion="always">
        {agents.map(agent => (
          <AgentSidebarGroup key={agent.name} agent={agent} />
        ))}
      </UIProvider>
    );

    expect(screen.getByTestId("agent-sidebar-group-agent-1")).toBeInTheDocument();
    expect(screen.getByTestId("agent-sidebar-group-agent-2")).toBeInTheDocument();
    expect(screen.getByTestId("agent-sidebar-group-agent-3")).toBeInTheDocument();
  });

  it('shows "No sessions" placeholder when children are empty', () => {
    renderGroup();
    expect(screen.getByTestId("agent-sidebar-group-empty-claude-agent")).toHaveTextContent(
      "No sessions"
    );
  });

  it("renders children instead of the empty placeholder", () => {
    renderGroup({}, <li data-testid="child-session">Session A</li>);
    expect(screen.getByTestId("child-session")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-sidebar-group-empty-claude-agent")).not.toBeInTheDocument();
  });

  it("collapses the content when the trigger is clicked", async () => {
    const user = userEvent.setup();
    renderGroup({}, <li data-testid="child-session">Session A</li>);

    const trigger = screen.getByTestId("agent-sidebar-group-trigger-claude-agent");
    expect(trigger).toHaveAttribute("aria-expanded", "true");

    await user.click(trigger);

    expect(trigger).toHaveAttribute("aria-expanded", "false");
  });

  it("renders a session count mono badge when sessionCount > 0", () => {
    renderGroup({ sessionCount: 3 });
    expect(screen.getByTestId("agent-sidebar-group-count-claude-agent")).toHaveTextContent("3");
  });

  it("hides the session count mono badge when sessionCount is 0", () => {
    renderGroup({ sessionCount: 0 });
    expect(screen.queryByTestId("agent-sidebar-group-count-claude-agent")).not.toBeInTheDocument();
  });
});
