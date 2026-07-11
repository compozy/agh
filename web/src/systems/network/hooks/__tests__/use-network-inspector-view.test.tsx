// @vitest-environment jsdom

import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  inspectorOpen: false,
  useChannelMembers: vi.fn(() => ({ members: [] })),
  useNetworkDirects: vi.fn(() => ({ directs: [] })),
  useNetworkThreads: vi.fn(() => ({ threads: [] })),
}));

vi.mock("../use-channel-members", () => ({
  useChannelMembers: mocks.useChannelMembers,
}));
vi.mock("../use-directs", () => ({
  useNetworkDirects: mocks.useNetworkDirects,
}));
vi.mock("../use-threads", () => ({
  useNetworkThreads: mocks.useNetworkThreads,
}));
vi.mock("../use-inspector-state", () => ({
  useInspectorState: () => ({
    close: vi.fn(),
    open: mocks.inspectorOpen,
    openWith: vi.fn(),
    setTab: vi.fn(),
    tab: "members",
    toggle: vi.fn(),
  }),
}));

import { useNetworkInspectorView } from "../use-network-inspector-view";

describe("useNetworkInspectorView", () => {
  beforeEach(() => {
    mocks.inspectorOpen = false;
    vi.clearAllMocks();
  });

  it("Should keep every inspector query disabled while the inspector is closed", () => {
    renderHook(() =>
      useNetworkInspectorView({ workspaceId: "ws_route", channel: "ops", enabled: true })
    );

    expect(mocks.useChannelMembers).toHaveBeenCalledWith("ops", {
      enabled: false,
      workspaceId: "ws_route",
    });
    expect(mocks.useNetworkThreads).toHaveBeenCalledWith("ops", {
      enabled: false,
      workspaceId: "ws_route",
    });
    expect(mocks.useNetworkDirects).toHaveBeenCalledWith("ops", {
      enabled: false,
      workspaceId: "ws_route",
    });
  });

  it("Should enable every inspector query only when both the rail and inspector are open", () => {
    mocks.inspectorOpen = true;
    renderHook(() =>
      useNetworkInspectorView({ workspaceId: "ws_route", channel: "ops", enabled: true })
    );

    expect(mocks.useChannelMembers).toHaveBeenCalledWith("ops", {
      enabled: true,
      workspaceId: "ws_route",
    });
    expect(mocks.useNetworkThreads).toHaveBeenCalledWith("ops", {
      enabled: true,
      workspaceId: "ws_route",
    });
    expect(mocks.useNetworkDirects).toHaveBeenCalledWith("ops", {
      enabled: true,
      workspaceId: "ws_route",
    });
  });
});
