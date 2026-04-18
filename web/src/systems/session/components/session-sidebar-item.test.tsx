import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { SessionPayload } from "../types";

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    params,
    ...rest
  }: {
    children?: React.ReactNode;
    to: string;
    params?: Record<string, string>;
    [key: string]: unknown;
  }) => (
    <a href={`${to}`.replace("$id", params?.id ?? "")} {...rest}>
      {children}
    </a>
  ),
  useMatchRoute: () => {
    return ({ params }: { params: { id: string } }) => {
      return params.id === "sess-active";
    };
  },
}));

vi.mock("@agh/ui", async importActual => {
  const actual = await importActual<typeof import("@agh/ui")>();
  return {
    ...actual,
    Badge: ({
      children,
      className,
      ...props
    }: {
      children: React.ReactNode;
      className?: string;
      variant?: string;
    }) => (
      <span data-testid="badge" className={className} {...props}>
        {children}
      </span>
    ),
  };
});

import { SessionSidebarItem } from "./session-sidebar-item";

const baseSession: SessionPayload = {
  id: "sess-001",
  name: "Test Session",
  agent_name: "claude-agent",
  workspace_id: "ws_alpha",
  workspace_path: "/tmp",
  state: "active",
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T01:00:00Z",
};

describe("SessionSidebarItem", () => {
  it("renders session title", () => {
    render(<SessionSidebarItem session={baseSession} />);
    expect(screen.getByText("Test Session")).toBeInTheDocument();
  });

  it("renders truncated ID when name is not set", () => {
    const session = { ...baseSession, name: undefined };
    render(<SessionSidebarItem session={session} />);
    expect(screen.getByText("sess-001")).toBeInTheDocument();
  });

  it("renders active badge for active session", () => {
    render(<SessionSidebarItem session={baseSession} />);
    expect(screen.getByText("active")).toBeInTheDocument();
  });

  it("renders workspace name and id metadata", () => {
    render(<SessionSidebarItem session={baseSession} workspaceName="alpha" />);

    expect(screen.getByTestId("workspace-name-badge")).toHaveTextContent("alpha");
    expect(screen.getByTestId("workspace-id-text")).toHaveTextContent("ws_alpha");
  });

  it("renders stopped badge for stopped session", () => {
    render(<SessionSidebarItem session={{ ...baseSession, state: "stopped" }} />);
    expect(screen.getByText("stopped")).toBeInTheDocument();
  });

  it("renders starting badge with pulse for starting session", () => {
    render(<SessionSidebarItem session={{ ...baseSession, state: "starting" }} />);
    expect(screen.getByText("starting")).toBeInTheDocument();
    const badge = screen.getByTestId("badge");
    expect(badge.className).toContain("animate-pulse");
  });

  it("renders stopping badge for stopping session", () => {
    render(<SessionSidebarItem session={{ ...baseSession, state: "stopping" }} />);
    expect(screen.getByText("stopping")).toBeInTheDocument();
  });

  it("navigates to /session/:id via Link", () => {
    render(<SessionSidebarItem session={baseSession} />);
    const button = screen.getByTestId("sidebar-sub-button");
    expect(button).toHaveAttribute("href", "/session/sess-001");
  });

  it("marks as active when route matches session id", () => {
    render(<SessionSidebarItem session={{ ...baseSession, id: "sess-active" }} />);
    const button = screen.getByTestId("sidebar-sub-button");
    expect(button).toHaveAttribute("data-active", "true");
  });

  it("marks as inactive when route does not match", () => {
    render(<SessionSidebarItem session={{ ...baseSession, id: "sess-other" }} />);
    const button = screen.getByTestId("sidebar-sub-button");
    expect(button).toHaveAttribute("data-active", "false");
  });

  it("shows amber dot when session has pending permission", () => {
    render(<SessionSidebarItem session={baseSession} hasPendingPermission={true} />);
    const indicator = screen.getByTestId("permission-indicator");
    expect(indicator).toBeInTheDocument();
    expect(indicator.className).toContain("animate-pulse");
    expect(indicator.className).toContain("bg-amber-500");
  });

  it("does not show amber dot when no pending permission", () => {
    render(<SessionSidebarItem session={baseSession} hasPendingPermission={false} />);
    expect(screen.queryByTestId("permission-indicator")).not.toBeInTheDocument();
  });

  it("does not show amber dot by default (undefined)", () => {
    render(<SessionSidebarItem session={baseSession} />);
    expect(screen.queryByTestId("permission-indicator")).not.toBeInTheDocument();
  });
});
