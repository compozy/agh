import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { NetworkActiveRoom, NetworkStatus } from "../types";
import { NetworkWorkspaceShell } from "./network-workspace-shell";

const noop = () => {};

function renderWorkspaceShell(statusOverrides: Partial<NetworkStatus> = {}) {
  const status: NetworkStatus = {
    channels: 0,
    delivery_workers: 0,
    enabled: true,
    local_peers: 0,
    messages_sent: 0,
    queued_messages: 0,
    remote_peers: 0,
    status: "running",
    ...statusOverrides,
  };

  return render(
    <NetworkWorkspaceShell
      activeKind="all"
      activeRoom={null}
      channelRooms={[]}
      composeDraft=""
      detailsTab="about"
      directRooms={[]}
      isComposePending={false}
      isDetailsOpen={false}
      isRoomLoading={false}
      isTimelineLoading={false}
      onComposeDraftChange={noop}
      onComposeSubmit={noop}
      onOpenCreateDialog={noop}
      onSelectDetailsTab={noop}
      onSelectKind={noop}
      onSelectRoom={noop}
      onSidebarQueryChange={noop}
      onToggleDetails={noop}
      onTogglePresence={noop}
      onToggleStarChannel={noop}
      roomError={null}
      selectedRoomKey={null}
      showPresence={false}
      sidebarQuery=""
      starredChannelRooms={[]}
      status={status}
    />
  );
}

describe("NetworkWorkspaceShell", () => {
  it.each([
    ["running", "success"],
    ["degraded", "warning"],
    ["stopped", "danger"],
  ] as const)("maps %s status to a %s header status dot", (status, tone) => {
    const { container } = renderWorkspaceShell({ status });

    const statusDot = container.querySelector('[data-slot="status-dot"]');

    expect(statusDot).not.toBeNull();
    expect(statusDot).toHaveAttribute("data-tone", tone);
  });

  it("surfaces the presence toggle as a separate control from the kind filters", () => {
    const activeRoom: NetworkActiveRoom = {
      aboutFields: [],
      canCompose: false,
      canStar: false,
      capabilities: [],
      channel: "builders",
      composeHint: null,
      composePlaceholder: "Send",
      description: "Presence history for #builders.",
      id: "builders",
      introBody: "Presence episodes are hidden from the default timeline.",
      introTitle: "Welcome to #builders",
      isStarred: false,
      key: "channel:builders",
      kindCounts: [],
      lastActivityAt: null,
      lastPresenceAt: "2026-04-13T10:40:00Z",
      memberCount: 0,
      members: [],
      messageCount: 0,
      messages: [],
      presenceCount: 12,
      preview: "Presence only",
      purpose: null,
      roomType: "channel",
      subtitle: "2 participants · 12 presence",
      title: "builders",
      wireFields: [],
    };

    const { getByRole } = render(
      <NetworkWorkspaceShell
        activeKind="all"
        activeRoom={activeRoom}
        channelRooms={[]}
        composeDraft=""
        detailsTab="about"
        directRooms={[]}
        isComposePending={false}
        isDetailsOpen={false}
        isRoomLoading={false}
        isTimelineLoading={false}
        onComposeDraftChange={noop}
        onComposeSubmit={noop}
        onOpenCreateDialog={noop}
        onSelectDetailsTab={noop}
        onSelectKind={noop}
        onSelectRoom={noop}
        onSidebarQueryChange={noop}
        onToggleDetails={noop}
        onTogglePresence={noop}
        onToggleStarChannel={noop}
        roomError={null}
        selectedRoomKey={null}
        showPresence={true}
        sidebarQuery=""
        starredChannelRooms={[]}
        status={{
          channels: 0,
          delivery_workers: 0,
          enabled: true,
          local_peers: 0,
          messages_sent: 0,
          queued_messages: 0,
          remote_peers: 0,
          status: "running",
        }}
      />
    );

    expect(getByRole("button", { name: /presence 12/i })).toHaveAttribute("aria-pressed", "true");
  });
});
