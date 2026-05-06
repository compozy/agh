// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    params,
    children,
    ...rest
  }: {
    to: string;
    params?: Record<string, string>;
    children: ReactNode;
    [key: string]: unknown;
  }) => {
    const path = Object.entries(params ?? {}).reduce(
      (acc, [key, value]) => acc.replace(`$${key}`, String(value)),
      to
    );
    return (
      <a href={path} {...(rest as Record<string, unknown>)}>
        {children}
      </a>
    );
  },
}));

import { ThreadsList } from "../threads-list";
import type { NetworkThreadSummary } from "../../../types";

const threads: NetworkThreadSummary[] = [
  {
    channel: "ops",
    last_activity_at: "2026-04-17T18:00:00Z",
    last_message_preview: "Pricing decision pending.",
    message_count: 12,
    open_work_count: 0,
    opened_at: "2026-04-17T17:00:00Z",
    opened_by_peer_id: "peer-codex",
    opened_session_id: "sess-1",
    participant_count: 3,
    root_message_id: "msg-1",
    thread_id: "thread-1",
    title: "Launch pricing decision",
  },
];

describe("ThreadsList", () => {
  it("Should render rows with title, preview, and metadata", () => {
    render(<ThreadsList activeThreadId={null} channel="ops" isLoading={false} threads={threads} />);

    expect(screen.getByTestId("network-thread-list-row-thread-1")).toBeInTheDocument();
    expect(screen.getByText("Launch pricing decision")).toBeInTheDocument();
    expect(screen.getByText("Pricing decision pending.")).toBeInTheDocument();
  });

  it("Should mark the active thread row with aria-current", () => {
    render(
      <ThreadsList activeThreadId="thread-1" channel="ops" isLoading={false} threads={threads} />
    );

    const row = screen.getByTestId("network-thread-list-row-thread-1");
    expect(row).toHaveAttribute("aria-current", "page");
  });

  it("Should render the loading skeleton when no threads are available yet", () => {
    render(<ThreadsList activeThreadId={null} channel="ops" isLoading threads={[]} />);
    expect(screen.getByTestId("network-thread-list-skeleton")).toBeInTheDocument();
  });

  it("Should render the empty state when no threads exist", () => {
    render(<ThreadsList activeThreadId={null} channel="ops" isLoading={false} threads={[]} />);
    expect(screen.getByText("No threads yet.")).toBeInTheDocument();
  });

  it("Should reduce contrast when dim is true", () => {
    render(
      <ThreadsList
        activeThreadId="thread-1"
        channel="ops"
        dim
        isLoading={false}
        threads={threads}
      />
    );
    expect(screen.getByTestId("network-thread-list")).toHaveAttribute("data-dim", "true");
  });
});
