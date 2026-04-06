import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { SessionPayload } from "../types";

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@/components/ui/badge", () => ({
  Badge: ({
    children,
    className,
    ...props
  }: {
    children: React.ReactNode;
    className?: string;
    variant?: string;
  }) => (
    <span data-testid="session-state-badge" className={className} {...props}>
      {children}
    </span>
  ),
}));

vi.mock("@/components/ui/button", () => ({
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
  it("shows session name and agent name", () => {
    const onStop = vi.fn();
    const onResume = vi.fn();
    render(<ChatHeader session={baseSession} onStop={onStop} onResume={onResume} />);

    expect(screen.getByText("My Test Session")).toBeInTheDocument();
    expect(screen.getByText("claude-code")).toBeInTheDocument();
  });

  it("shows workspace metadata", () => {
    render(
      <ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} workspaceName="alpha" />
    );

    expect(screen.getByTestId("session-workspace-badge")).toHaveTextContent("alpha");
    expect(screen.getByTestId("session-workspace-id")).toHaveTextContent("ws_alpha");
  });

  it("shows session ID when name is not set", () => {
    const session = { ...baseSession, name: undefined };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} />);

    expect(screen.getByText("sess-001")).toBeInTheDocument();
  });

  it("shows correct state badge", () => {
    render(<ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} />);

    const badge = screen.getByTestId("session-state-badge");
    expect(badge).toHaveTextContent("active");
  });

  it("shows starting badge with pulse animation", () => {
    const session = { ...baseSession, state: "starting" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} />);

    const badge = screen.getByTestId("session-state-badge");
    expect(badge).toHaveTextContent("starting");
    expect(badge.className).toContain("animate-pulse");
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
