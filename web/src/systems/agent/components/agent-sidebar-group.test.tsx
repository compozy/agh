import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "../types";

vi.mock("@/components/ui/collapsible", () => ({
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
}));

vi.mock("@/components/ui/sidebar", () => ({
  SidebarGroup: ({ children, ...props }: { children: React.ReactNode }) => (
    <div data-testid="sidebar-group" {...props}>
      {children}
    </div>
  ),
  SidebarGroupLabel: ({
    children,
    render: _render,
    ...props
  }: {
    children: React.ReactNode;
    render?: unknown;
  }) => (
    <div data-testid="sidebar-group-label" {...props}>
      {children}
    </div>
  ),
  SidebarGroupAction: ({
    children,
    ...props
  }: {
    children: React.ReactNode;
    title?: string;
    onClick?: () => void;
  }) => (
    <button data-testid="sidebar-group-action" {...props}>
      {children}
    </button>
  ),
  SidebarGroupContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sidebar-group-content">{children}</div>
  ),
  SidebarMenu: ({ children }: { children: React.ReactNode }) => (
    <ul data-testid="sidebar-menu">{children}</ul>
  ),
  SidebarMenuButton: ({
    children,
    tooltip: _tooltip,
    ...props
  }: {
    children: React.ReactNode;
    tooltip?: string;
  }) => (
    <button data-testid="sidebar-menu-button" {...props}>
      {children}
    </button>
  ),
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => (
    <li data-testid="sidebar-menu-item">{children}</li>
  ),
  SidebarMenuSub: ({ children }: { children: React.ReactNode }) => (
    <ul data-testid="sidebar-menu-sub">{children}</ul>
  ),
}));

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

  it('shows "New Session" button', () => {
    render(<AgentSidebarGroup agent={mockAgent} />);
    expect(screen.getByText("New Session")).toBeInTheDocument();
  });

  it("calls onNewSession with agent name when clicking New Session", () => {
    const onNewSession = vi.fn();
    render(<AgentSidebarGroup agent={mockAgent} onNewSession={onNewSession} />);
    screen.getByTestId("sidebar-group-action").click();
    expect(onNewSession).toHaveBeenCalledWith("claude-agent");
  });

  it("renders one group per agent from mock data", () => {
    const agents: AgentPayload[] = [
      { name: "agent-1", provider: "claude", prompt: "prompt1" },
      { name: "agent-2", provider: "codex", prompt: "prompt2" },
      { name: "agent-3", provider: "gemini", prompt: "prompt3" },
    ];

    const { container } = render(
      <>
        {agents.map(agent => (
          <AgentSidebarGroup key={agent.name} agent={agent} />
        ))}
      </>
    );

    expect(screen.getByText("agent-1")).toBeInTheDocument();
    expect(screen.getByText("agent-2")).toBeInTheDocument();
    expect(screen.getByText("agent-3")).toBeInTheDocument();
    expect(container.querySelectorAll('[data-testid="sidebar-group"]')).toHaveLength(3);
  });

  it('shows "No sessions" placeholder text', () => {
    render(<AgentSidebarGroup agent={mockAgent} />);
    expect(screen.getByText("No sessions")).toBeInTheDocument();
  });
});
