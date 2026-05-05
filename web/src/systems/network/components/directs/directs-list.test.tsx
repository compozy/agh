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

import { DirectsList } from "./directs-list";
import type { NetworkDirectRoomSummary } from "../../types";

const directs: NetworkDirectRoomSummary[] = [
  {
    channel: "ops",
    direct_id: "direct-1",
    last_activity_at: "2026-04-17T18:00:00Z",
    last_message_preview: "Confirmed.",
    message_count: 4,
    open_work_count: 0,
    opened_at: "2026-04-17T17:00:00Z",
    peer_a: "peer-self",
    peer_b: "peer-remote",
  },
];

describe("DirectsList", () => {
  it("Should render the row labelled with the OTHER party (peer_a/peer_b lex order is invisible)", () => {
    render(
      <DirectsList
        activeDirectId={null}
        channel="ops"
        directs={directs}
        isLoading={false}
        selfPeerId="peer-self"
      />
    );

    expect(screen.getByText("@peer-remote")).toBeInTheDocument();
    expect(screen.queryByText("@peer-self")).toBeNull();
  });

  it("Should render the empty state when no directs exist", () => {
    render(<DirectsList activeDirectId={null} channel="ops" directs={[]} isLoading={false} />);
    expect(screen.getByText("No direct rooms yet.")).toBeInTheDocument();
  });

  it("Should render the loading skeleton when no directs are available yet", () => {
    render(<DirectsList activeDirectId={null} channel="ops" directs={[]} isLoading />);
    expect(screen.getByTestId("network-direct-list-skeleton")).toBeInTheDocument();
  });
});
