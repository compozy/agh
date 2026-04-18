import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "../types";

vi.mock("@agh/ui", async importActual => {
  const actual = await importActual<typeof import("@agh/ui")>();
  return {
    ...actual,
    Collapsible: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="collapsible">{children}</div>
    ),
    CollapsibleTrigger: ({ children, ...props }: { children?: React.ReactNode }) => (
      <button data-testid="collapsible-trigger" {...props}>
        {children}
      </button>
    ),
    CollapsibleContent: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="collapsible-content">{children}</div>
    ),
  };
});

import { AgentSidebarGroup } from "./agent-sidebar-group";

const mockAgent: AgentPayload = {
  name: "claude-agent",
  provider: "claude",
  prompt: "You are a helpful assistant",
};

describe("AgentSidebarGroup", () => {
  it("renders agent name", () => {
    render(<AgentSidebarGroup agent={mockAgent} />);
    expect(screen.getByText("claude-agent")).toBeInTheDocument();
  });

  it("shows a New Session action", () => {
    render(<AgentSidebarGroup agent={mockAgent} />);
    expect(screen.getByRole("button", { name: "New Session" })).toBeInTheDocument();
  });

  it("calls onNewSession with agent name when clicking New Session", () => {
    const onNewSession = vi.fn();
    render(<AgentSidebarGroup agent={mockAgent} onNewSession={onNewSession} />);
    screen.getByRole("button", { name: "New Session" }).click();
    expect(onNewSession).toHaveBeenCalledWith("claude-agent");
  });

  it("disables the new-session action when requested", () => {
    const onNewSession = vi.fn();
    render(
      <AgentSidebarGroup agent={mockAgent} onNewSession={onNewSession} newSessionDisabled={true} />
    );

    const action = screen.getByRole("button", { name: "New Session" });
    expect(action).toBeDisabled();
    action.click();
    expect(onNewSession).not.toHaveBeenCalled();
  });

  it("renders one group per agent from mock data", () => {
    const agents: AgentPayload[] = [
      { name: "agent-1", provider: "claude", prompt: "prompt1" },
      { name: "agent-2", provider: "codex", prompt: "prompt2" },
      { name: "agent-3", provider: "gemini", prompt: "prompt3" },
    ];

    render(
      <>
        {agents.map(agent => (
          <AgentSidebarGroup key={agent.name} agent={agent} />
        ))}
      </>
    );

    expect(screen.getByText("agent-1")).toBeInTheDocument();
    expect(screen.getByText("agent-2")).toBeInTheDocument();
    expect(screen.getByText("agent-3")).toBeInTheDocument();
    expect(screen.getAllByTestId("collapsible")).toHaveLength(3);
  });

  it('shows "No sessions" placeholder text', () => {
    render(<AgentSidebarGroup agent={mockAgent} />);
    expect(screen.getByText("No sessions")).toBeInTheDocument();
  });
});
