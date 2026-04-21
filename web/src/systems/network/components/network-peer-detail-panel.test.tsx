import { render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { networkPeerFixture } from "../mocks";
import type { NetworkPeerDetail } from "../types";
import { NetworkPeerDetailPanel } from "./network-peer-detail-panel";

function renderPanel(
  options: {
    error?: Error | null;
    isLoading?: boolean;
    peer?: NetworkPeerDetail;
  } = {}
) {
  return render(
    <NetworkPeerDetailPanel
      error={options.error ?? null}
      isLoading={options.isLoading ?? false}
      peer={options.peer === undefined ? networkPeerFixture : options.peer}
    />
  );
}

describe("NetworkPeerDetailPanel", () => {
  it("Should render exactly one Metric per stat key and one row per unified capability", () => {
    renderPanel();

    const panel = screen.getByTestId("network-peer-detail-panel");
    const metrics = within(panel).getAllByTestId(/^network-peer-metric-/);
    expect(metrics).toHaveLength(4);

    const capabilityRoots = within(panel).getAllByTestId(/^network-peer-capability-[^-]+$/);
    expect(capabilityRoots).toHaveLength(networkPeerFixture.peer_card.capabilities.length);
  });

  it("Should render brief summary and rich catalog fields for each capability", () => {
    renderPanel();

    const panel = screen.getByTestId("network-peer-detail-panel");
    const chatRow = within(panel).getByTestId("network-peer-capability-chat");
    expect(within(chatRow).getByTestId("network-peer-capability-chat-summary")).toHaveTextContent(
      "Coordinates chat-first collaboration."
    );
    expect(within(chatRow).getByTestId("network-peer-capability-chat-outcome")).toHaveTextContent(
      "Peers converge on a shared plan"
    );
    expect(
      within(chatRow).getByTestId("network-peer-capability-chat-execution-outline")
    ).toBeInTheDocument();

    const toolsRow = within(panel).getByTestId("network-peer-capability-tools");
    const requires = within(toolsRow).getByTestId("network-peer-capability-tools-requirements");
    expect(requires).toHaveTextContent("chat");
    expect(within(toolsRow).getByTestId("network-peer-capability-tools-version")).toHaveTextContent(
      "v0.2.0"
    );
  });

  it("Should fall back to a brief label when no rich capability catalog is supplied", () => {
    renderPanel({
      peer: {
        ...networkPeerFixture,
        capability_catalog: null,
      },
    });

    const panel = screen.getByTestId("network-peer-detail-panel");
    expect(panel).toHaveTextContent("brief");
    expect(
      within(panel).queryByTestId("network-peer-capability-chat-detail")
    ).not.toBeInTheDocument();
  });

  it("Should render the Message Statistics section header, channel table, and session link", () => {
    renderPanel();

    const panel = screen.getByTestId("network-peer-detail-panel");
    expect(panel).toHaveTextContent("Message Statistics");
    expect(panel).toHaveTextContent("View Session");
    expect(
      within(panel).getByTestId(`network-peer-channel-${networkPeerFixture.channel}`)
    ).toBeInTheDocument();
  });

  it("Should show the loading fallback when isLoading=true", () => {
    renderPanel({ isLoading: true });
    expect(screen.getByTestId("network-peer-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("network-peer-detail-panel")).toBeNull();
  });

  it("Should surface the error Empty state when an error is present", () => {
    renderPanel({
      error: new Error("Peer not found"),
      peer: undefined as unknown as NetworkPeerDetail,
    });

    const error = screen.getByTestId("network-peer-error");
    expect(error).toHaveTextContent("Unable to load peer");
    expect(error).toHaveTextContent("Peer not found");
  });
});
