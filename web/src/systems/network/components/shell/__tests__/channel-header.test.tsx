// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
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

vi.mock("../../../hooks/use-channel-members", () => ({
  useChannelMembers: () => ({
    members: [],
    agentCount: 1,
    humanCount: 1,
    isLoading: false,
  }),
}));

import { ChannelHeader } from "../channel-header";
import type { NetworkChannel, NetworkChannelSummary } from "@/systems/network";

function renderHeader({
  channel = sampleChannel,
  detail = null,
  inspectorOpen = false,
}: {
  channel?: NetworkChannelSummary;
  detail?: NetworkChannel | null;
  inspectorOpen?: boolean;
} = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(
    <ChannelHeader
      channel={channel}
      detail={detail}
      inspectorOpen={inspectorOpen}
      onInspectorToggle={() => undefined}
      openWorkCount={2}
    />,
    { wrapper }
  );
}

const sampleChannel: NetworkChannelSummary = {
  channel: "ops",
  workspace_id: "w1",
  created_at: "2026-04-17T14:00:00Z",
  created_by: "ops",
  peer_count: 4,
};

describe("ChannelHeader", () => {
  it("Should emit a <DetailHeader> 24 px H1", () => {
    renderHeader();
    const wrapper = screen.getByTestId("network-channel-header");
    const detailHeader = wrapper.querySelector('[data-slot="detail-header"]');
    expect(detailHeader).not.toBeNull();
    const title = wrapper.querySelector('[data-slot="detail-header-title"]');
    expect(title).not.toBeNull();
    expect(screen.getByTestId("network-channel-title")).toHaveTextContent("ops");
  });

  it("Should NOT render the channel-search button", () => {
    renderHeader();
    expect(screen.queryByTestId("network-channel-search")).toBeNull();
    expect(screen.queryByRole("button", { name: /search/i })).toBeNull();
  });

  it("Should expose the inspector toggle and kebab actions in the DetailHeader actions slot", () => {
    renderHeader();
    expect(screen.getByTestId("network-channel-inspector-toggle")).toBeInTheDocument();
    expect(screen.getByTestId("network-channel-kebab")).toBeInTheDocument();
  });
});
