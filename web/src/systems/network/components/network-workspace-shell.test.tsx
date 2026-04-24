import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { NetworkStatus } from "../types";
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
      onToggleStarChannel={noop}
      roomError={null}
      selectedRoomKey={null}
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
});
