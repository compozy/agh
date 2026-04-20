import { render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { networkChannelFixture, networkChannelMessagesFixture } from "../mocks";
import type { NetworkChannel, NetworkChannelMessage } from "../types";
import { NetworkChannelDetailPanel } from "./network-channel-detail-panel";

interface RenderOptions {
  channel?: NetworkChannel;
  error?: Error | null;
  isLoading?: boolean;
  isMessagesLoading?: boolean;
  messages?: NetworkChannelMessage[];
}

function renderPanel(options: RenderOptions = {}) {
  return render(
    <NetworkChannelDetailPanel
      channel={options.channel === undefined ? networkChannelFixture : options.channel}
      error={options.error ?? null}
      isLoading={options.isLoading ?? false}
      isMessagesLoading={options.isMessagesLoading ?? false}
      messages={options.messages ?? networkChannelMessagesFixture}
    />
  );
}

describe("NetworkChannelDetailPanel", () => {
  it("Should render one CodeBlock per message payload and one KindChip per message", () => {
    renderPanel();

    const panel = screen.getByTestId("network-channel-detail-panel");
    const payloads = within(panel).getAllByTestId(/^network-channel-message-payload-/);
    expect(payloads).toHaveLength(networkChannelMessagesFixture.length);
    expect(within(panel).getAllByTestId(/^network-channel-message-kind-/)).toHaveLength(
      networkChannelMessagesFixture.length
    );
  });

  it("Should render the wire trace Table and members list alongside the messages section", () => {
    renderPanel();

    const panel = screen.getByTestId("network-channel-detail-panel");
    expect(within(panel).getByTestId("network-channel-wire-trace")).toBeInTheDocument();
    expect(within(panel).getByTestId("network-channel-members-list")).toBeInTheDocument();
    expect(within(panel).getAllByTestId(/^network-channel-member-/)).toHaveLength(
      networkChannelFixture.peers?.length ?? 0
    );
  });

  it("Should render the loading skeleton instead of the detail body when isLoading=true", () => {
    renderPanel({ isLoading: true });

    expect(screen.getByTestId("network-channel-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("network-channel-detail-panel")).toBeNull();
  });

  it("Should render the messages-loading fallback when messages are still fetching", () => {
    renderPanel({ isMessagesLoading: true, messages: [] });

    expect(screen.getByTestId("network-channel-messages-loading")).toBeInTheDocument();
  });

  it("Should surface the error Empty state when an error is present", () => {
    renderPanel({
      channel: undefined as unknown as NetworkChannel,
      error: new Error("Load failed"),
    });

    const error = screen.getByTestId("network-channel-error");
    expect(error).toHaveTextContent("Unable to load channel");
    expect(error).toHaveTextContent("Load failed");
  });
});
