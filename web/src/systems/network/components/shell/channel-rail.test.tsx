// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

import { ChannelRail } from "./channel-rail";
import type { NetworkChannelSummary, NetworkRecentEntry } from "@/systems/network";

const channels: NetworkChannelSummary[] = [
  {
    channel: "alpha",
    workspace_id: "w1",
    created_at: "2026-04-17T14:00:00Z",
    created_by: "ops",
    peer_count: 2,
  },
  {
    channel: "design",
    workspace_id: "w1",
    created_at: "2026-04-17T14:00:00Z",
    created_by: "ops",
    peer_count: 2,
  },
  {
    channel: "ops",
    workspace_id: "w1",
    created_at: "2026-04-17T14:00:00Z",
    created_by: "ops",
    peer_count: 4,
    last_activity_at: "2026-04-17T18:00:00Z",
  },
];

const recents: NetworkRecentEntry[] = [
  {
    surface: "thread",
    channel: "ops",
    containerId: "thread_ops_one",
    preview: "Plan the migration",
    lastActivityAt: "2026-04-17T18:16:00Z",
    hasUnread: true,
    participantLabel: "4 peers",
  },
  {
    surface: "direct",
    channel: "design",
    containerId: "direct_design_one",
    preview: "ETA 18:30",
    lastActivityAt: "2026-04-17T17:00:00Z",
    hasUnread: false,
    participantLabel: "two-party",
  },
];

interface HarnessProps {
  pinnedIds?: string[];
  togglePinned?: (channel: string) => void;
}

function Harness({ pinnedIds = ["alpha"], togglePinned = () => undefined }: HarnessProps) {
  const pinnedSet = new Set(pinnedIds);
  return (
    <ChannelRail
      activeChannel="ops"
      activeDirectId={null}
      directs={[]}
      hasUnread={() => true}
      isChannelsLoading={false}
      isDirectsLoading={false}
      isPinned={channel => pinnedSet.has(channel)}
      isRecentsLoading={false}
      onTogglePinned={togglePinned}
      pinnedChannels={channels.filter(channel => pinnedSet.has(channel.channel))}
      recents={recents}
      selfPeerId={null}
      unpinnedChannels={channels.filter(channel => !pinnedSet.has(channel.channel))}
    />
  );
}

function renderRail(props: HarnessProps = {}) {
  return render(<Harness {...props} />);
}

describe("ChannelRail", () => {
  it("renders pinned channels above unpinned, alphabetical thereafter", () => {
    renderRail({ pinnedIds: ["alpha"] });

    const channelOrder = screen
      .getAllByRole("link")
      .filter(link => link.getAttribute("data-testid")?.startsWith("network-channel-link-"))
      .map(link => link.getAttribute("data-testid")?.replace("network-channel-link-", "") ?? "");

    expect(channelOrder).toEqual(["alpha", "design", "ops"]);
  });

  it("renders cross-channel recents with surface-specific icons", () => {
    renderRail();
    const threadRecent = screen.getByTestId("network-recents-thread-thread_ops_one");
    const directRecent = screen.getByTestId("network-recents-direct-direct_design_one");
    expect(threadRecent).toBeDefined();
    expect(directRecent).toBeDefined();
    expect(threadRecent.querySelector("[aria-label='Thread']")).not.toBeNull();
    expect(directRecent.querySelector("[aria-label='Direct room']")).not.toBeNull();
  });

  it("invokes togglePinned when the pin affordance is clicked", async () => {
    const togglePinned = vi.fn();
    renderRail({ pinnedIds: [], togglePinned });
    const user = userEvent.setup();
    await user.click(screen.getByTestId("network-channel-pin-ops"));
    expect(togglePinned).toHaveBeenCalledWith("ops");
  });

  it("renders empty-state copy when no channels are visible", () => {
    render(
      <ChannelRail
        activeChannel={null}
        activeDirectId={null}
        directs={[]}
        hasUnread={() => false}
        isChannelsLoading={false}
        isDirectsLoading={false}
        isPinned={() => false}
        isRecentsLoading={false}
        onTogglePinned={() => undefined}
        pinnedChannels={[]}
        recents={[]}
        selfPeerId={null}
        unpinnedChannels={[]}
      />
    );
    expect(screen.getByTestId("network-channels-empty")).toHaveTextContent("No channels yet.");
  });
});
