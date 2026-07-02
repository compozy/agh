// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ChannelPolicyDialog } from "../channel-policy-dialog";
import type { ChannelMember } from "../../../hooks/use-channel-members";
import type { NetworkChannel, NetworkChannelSummary } from "../../../types";

const channel: NetworkChannelSummary = {
  channel: "ops",
  coordinator_peer_id: "peer-release",
  created_at: "2026-04-17T14:00:00Z",
  created_by: "ops",
  fanout_policy: "coordinator",
  peer_count: 4,
  purpose: "Coordinate launch.",
  workspace_id: "ws_1",
};

const detail = {
  ...channel,
  kind_counts: [],
  peers: [],
} as NetworkChannel;

const members: ChannelMember[] = [
  {
    displayName: "Release",
    lastSeenAgeSeconds: null,
    local: true,
    peerId: "peer-release",
    presenceState: "local",
    role: "agent",
  },
];

describe("ChannelPolicyDialog", () => {
  it("Should submit purpose, policy, and coordinator peer", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const onOpenChange = vi.fn();

    render(
      <ChannelPolicyDialog
        channel={channel}
        detail={detail}
        isSubmitting={false}
        members={members}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
        open
      />
    );

    await user.clear(screen.getByTestId("channel-policy-purpose"));
    await user.type(screen.getByTestId("channel-policy-purpose"), "Coordinate final rollout.");
    await user.click(screen.getByTestId("channel-policy-submit"));

    expect(onSubmit).toHaveBeenCalledWith({
      coordinator_peer_id: "peer-release",
      fanout_policy: "coordinator",
      purpose: "Coordinate final rollout.",
    });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("Should reset the coordinator peer when selecting a non-coordinator fanout policy", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const onOpenChange = vi.fn();

    render(
      <ChannelPolicyDialog
        channel={channel}
        detail={detail}
        isSubmitting={false}
        members={members}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
        open
      />
    );

    await user.click(screen.getByTestId("channel-policy-fanout"));
    await user.click(await screen.findByText("All members"));
    await user.click(screen.getByTestId("channel-policy-submit"));

    expect(onSubmit).toHaveBeenCalledWith({
      coordinator_peer_id: "",
      fanout_policy: "all_members",
      purpose: "Coordinate launch.",
    });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("Should keep the dialog open and report submit errors", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockRejectedValue(new Error("save failed"));
    const onOpenChange = vi.fn();

    render(
      <ChannelPolicyDialog
        channel={channel}
        detail={detail}
        isSubmitting={false}
        members={members}
        onOpenChange={onOpenChange}
        onSubmit={onSubmit}
        open
      />
    );

    await user.click(screen.getByTestId("channel-policy-submit"));

    expect(await screen.findByRole("alert")).toHaveTextContent("save failed");
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });
});
