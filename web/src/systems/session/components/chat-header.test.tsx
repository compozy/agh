import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { SessionPayload } from "../types";

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@agh/ui", () => ({
  Button: ({
    children,
    onClick,
    ...props
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    [key: string]: unknown;
  }) => (
    <button onClick={onClick} {...props}>
      {children}
    </button>
  ),
}));

import { ChatHeader } from "./chat-header";

const baseSession: SessionPayload = {
  id: "sess-001",
  name: "My Test Session",
  agent_name: "claude-code",
  workspace_id: "ws_alpha",
  workspace_path: "/tmp/workspace",
  state: "active",
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T01:00:00Z",
};

describe("ChatHeader", () => {
  it("renders breadcrumb with agent name and session name", () => {
    render(<ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} />);

    expect(screen.getByTestId("chat-breadcrumb")).toBeInTheDocument();
    expect(screen.getByText("claude-code")).toBeInTheDocument();
    expect(screen.getByTestId("session-name")).toHaveTextContent("My Test Session");
  });

  it("shows agent status dot with success color for active state", () => {
    render(<ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} />);

    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.className).toMatch(/bg-\[color:var\(--color-success\)\]/);
  });

  it("shows agent status dot with warning color and pulse for starting state", () => {
    const session = { ...baseSession, state: "starting" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} />);

    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.className).toMatch(/bg-\[color:var\(--color-warning\)\]/);
    expect(dot.className).toContain("animate-pulse");
  });

  it("shows agent status dot with tertiary color for stopped state", () => {
    const session = { ...baseSession, state: "stopped" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} />);

    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.className).toMatch(/bg-\[color:var\(--color-text-tertiary\)\]/);
  });

  it("shows workspace name in breadcrumb when provided", () => {
    render(
      <ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} workspaceName="alpha" />
    );

    expect(screen.getByTestId("session-workspace-badge")).toHaveTextContent("alpha");
  });

  it("shows session ID when name is not set", () => {
    const session = { ...baseSession, name: undefined };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} />);

    expect(screen.getByTestId("session-name")).toHaveTextContent("sess-001");
  });

  it("shows stop button for active session", () => {
    render(<ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} />);

    expect(screen.getByTestId("stop-button")).toBeInTheDocument();
    expect(screen.queryByTestId("resume-button")).not.toBeInTheDocument();
  });

  it("shows resume button for stopped session", () => {
    const session = { ...baseSession, state: "stopped" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} />);

    expect(screen.getByTestId("resume-button")).toBeInTheDocument();
    expect(screen.queryByTestId("stop-button")).not.toBeInTheDocument();
  });

  it("calls onStop when stop button is clicked", () => {
    const onStop = vi.fn();
    render(<ChatHeader session={baseSession} onStop={onStop} onResume={vi.fn()} />);

    fireEvent.click(screen.getByTestId("stop-button"));

    expect(onStop).toHaveBeenCalledOnce();
  });

  it("calls onResume when resume button is clicked", () => {
    const onResume = vi.fn();
    const session = { ...baseSession, state: "stopped" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={onResume} />);

    fireEvent.click(screen.getByTestId("resume-button"));

    expect(onResume).toHaveBeenCalledOnce();
  });

  it("hides both buttons during stopping state", () => {
    const session = { ...baseSession, state: "stopping" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} />);

    expect(screen.queryByTestId("stop-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("resume-button")).not.toBeInTheDocument();
  });
});
