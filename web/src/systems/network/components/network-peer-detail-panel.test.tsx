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
  it("Should render exactly one Metric per stat key and one KindChip per capability", () => {
    renderPanel();

    const panel = screen.getByTestId("network-peer-detail-panel");
    const metrics = within(panel).getAllByTestId(/^network-peer-metric-/);
    expect(metrics).toHaveLength(4);

    const capabilities = within(panel).getAllByTestId(/^network-peer-capability-/);
    expect(capabilities).toHaveLength(networkPeerFixture.peer_card.capabilities.length);
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
