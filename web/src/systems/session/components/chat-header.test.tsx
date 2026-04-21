import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { SessionPayload } from "../types";

vi.mock("@/lib/utils", async importActual => {
  const actual = await importActual<typeof import("@/lib/utils")>();
  return {
    ...actual,
    cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
  };
});

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
    render(
      <ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} onClear={vi.fn()} />
    );

    expect(screen.getByTestId("chat-breadcrumb")).toBeInTheDocument();
    expect(screen.getByText("claude-code")).toBeInTheDocument();
    expect(screen.getByTestId("session-name")).toHaveTextContent("My Test Session");
  });

  it("renders status dot with success tone for active state", () => {
    render(
      <ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} onClear={vi.fn()} />
    );

    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.getAttribute("data-slot")).toBe("status-dot");
    expect(dot.getAttribute("data-tone")).toBe("success");
    expect(dot.getAttribute("data-size")).toBe("md");
  });

  it("renders status dot with warning tone and pulse for starting state", () => {
    const session = { ...baseSession, state: "starting" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} onClear={vi.fn()} />);

    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.getAttribute("data-tone")).toBe("warning");
    expect(dot.getAttribute("data-pulse")).toBe("true");
  });

  it("renders status dot with neutral tone for stopped state", () => {
    const session = { ...baseSession, state: "stopped" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} onClear={vi.fn()} />);

    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.getAttribute("data-tone")).toBe("neutral");
  });

  it("shows workspace name in breadcrumb when provided", () => {
    render(
      <ChatHeader
        session={baseSession}
        onStop={vi.fn()}
        onResume={vi.fn()}
        onClear={vi.fn()}
        workspaceName="alpha"
      />
    );

    const badge = screen.getByTestId("session-workspace-badge");
    expect(badge).toHaveTextContent("alpha");
    expect(badge.getAttribute("data-slot")).toBe("mono-badge");
  });

  it("shows session ID when name is not set", () => {
    const session = { ...baseSession, name: undefined };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} onClear={vi.fn()} />);

    expect(screen.getByTestId("session-name")).toHaveTextContent("sess-001");
  });

  it("shows stop button for active session", () => {
    render(
      <ChatHeader session={baseSession} onStop={vi.fn()} onResume={vi.fn()} onClear={vi.fn()} />
    );

    expect(screen.getByTestId("stop-button")).toBeInTheDocument();
    expect(screen.queryByTestId("resume-button")).not.toBeInTheDocument();
  });

  it("shows resume button for stopped session", () => {
    const session = { ...baseSession, state: "stopped" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} onClear={vi.fn()} />);

    expect(screen.getByTestId("resume-button")).toBeInTheDocument();
    expect(screen.queryByTestId("stop-button")).not.toBeInTheDocument();
  });

  it("calls onStop when stop button is clicked", () => {
    const onStop = vi.fn();
    render(
      <ChatHeader session={baseSession} onStop={onStop} onResume={vi.fn()} onClear={vi.fn()} />
    );

    fireEvent.click(screen.getByTestId("stop-button"));

    expect(onStop).toHaveBeenCalledOnce();
  });

  it("calls onResume when resume button is clicked", () => {
    const onResume = vi.fn();
    const session = { ...baseSession, state: "stopped" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={onResume} onClear={vi.fn()} />);

    fireEvent.click(screen.getByTestId("resume-button"));

    expect(onResume).toHaveBeenCalledOnce();
  });

  it("opens a confirmation dialog before clearing the conversation", () => {
    const onClear = vi.fn();
    render(
      <ChatHeader
        session={baseSession}
        onStop={vi.fn()}
        onResume={vi.fn()}
        onClear={onClear}
        canClear
      />
    );

    fireEvent.click(screen.getByTestId("clear-button"));
    expect(screen.getByTestId("clear-dialog")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("clear-dialog-confirm"));
    expect(onClear).toHaveBeenCalledOnce();
  });

  it("disables the clear trigger when there is nothing to clear", () => {
    render(
      <ChatHeader
        session={baseSession}
        onStop={vi.fn()}
        onResume={vi.fn()}
        onClear={vi.fn()}
        canClear={false}
      />
    );

    expect(screen.getByTestId("clear-button")).toBeDisabled();
  });

  it("shows loading feedback on stop, resume, and clear controls", () => {
    const { rerender } = render(
      <ChatHeader
        session={baseSession}
        onStop={vi.fn()}
        onResume={vi.fn()}
        onClear={vi.fn()}
        canClear
        isStopping
      />
    );

    expect(screen.getByTestId("stop-button").querySelector("svg")).toHaveClass("animate-spin");

    rerender(
      <ChatHeader
        session={{ ...baseSession, state: "stopped" }}
        onStop={vi.fn()}
        onResume={vi.fn()}
        onClear={vi.fn()}
        canClear
        isResuming
      />
    );
    expect(screen.getByTestId("resume-button").querySelector("svg")).toHaveClass("animate-spin");

    rerender(
      <ChatHeader
        session={baseSession}
        onStop={vi.fn()}
        onResume={vi.fn()}
        onClear={vi.fn()}
        canClear
        isClearing
      />
    );
    expect(screen.getByTestId("clear-button").querySelector("svg")).toHaveClass("animate-spin");
  });

  it("hides both stop and resume buttons during stopping state", () => {
    const session = { ...baseSession, state: "stopping" as const };
    render(<ChatHeader session={session} onStop={vi.fn()} onResume={vi.fn()} onClear={vi.fn()} />);

    expect(screen.queryByTestId("stop-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("resume-button")).not.toBeInTheDocument();
  });
});
