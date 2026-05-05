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

import { ActivityFeed } from "./activity-feed";

describe("ActivityFeed", () => {
  it("Should sort entries by last_activity_at across both surfaces", () => {
    render(
      <ActivityFeed
        channel="ops"
        directs={[
          {
            channel: "ops",
            direct_id: "direct-1",
            last_activity_at: "2026-04-17T17:00:00Z",
            last_message_preview: "Older direct",
            message_count: 1,
            open_work_count: 0,
            opened_at: "2026-04-17T16:00:00Z",
            peer_a: "self",
            peer_b: "remote",
          },
        ]}
        isLoading={false}
        threads={[
          {
            channel: "ops",
            last_activity_at: "2026-04-17T18:00:00Z",
            last_message_preview: "Newer thread",
            message_count: 1,
            open_work_count: 0,
            opened_at: "2026-04-17T17:00:00Z",
            opened_by_peer_id: "self",
            opened_session_id: "sess-1",
            participant_count: 2,
            root_message_id: "m1",
            thread_id: "thread-1",
            title: "Newer thread",
          },
        ]}
      />
    );

    const entries = screen
      .getAllByTestId(/network-activity-entry-/u)
      .map(node => node.getAttribute("data-testid") ?? "");
    expect(entries[0]).toContain("thread:thread-1");
    expect(entries[1]).toContain("direct:direct-1");
  });

  it("Should render the kind tag prefixes [TH] and [DM]", () => {
    render(
      <ActivityFeed
        channel="ops"
        directs={[
          {
            channel: "ops",
            direct_id: "direct-1",
            last_activity_at: "2026-04-17T17:00:00Z",
            last_message_preview: "Direct preview",
            message_count: 1,
            open_work_count: 0,
            opened_at: "2026-04-17T16:00:00Z",
            peer_a: "self",
            peer_b: "remote",
          },
        ]}
        isLoading={false}
        threads={[
          {
            channel: "ops",
            last_activity_at: "2026-04-17T18:00:00Z",
            last_message_preview: "Thread preview",
            message_count: 1,
            open_work_count: 0,
            opened_at: "2026-04-17T17:00:00Z",
            opened_by_peer_id: "self",
            opened_session_id: "sess-1",
            participant_count: 2,
            root_message_id: "m1",
            thread_id: "thread-1",
            title: "Thread title",
          },
        ]}
      />
    );

    expect(screen.getByText("[TH]")).toBeInTheDocument();
    expect(screen.getByText("[DM]")).toBeInTheDocument();
  });
});
