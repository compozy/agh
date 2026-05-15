// Suite: Network route shell workspace synchronization
// Invariant: A stale Network detail URL must not overwrite an explicit sidebar workspace switch.
// Boundary IN: useNetworkRouteShell workspace/route synchronization behavior.
// Boundary OUT: Sidebar click rendering and browser integration, covered by app layout tests and QA Playwright evidence.

import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { mockNavigate, mockSetActiveWorkspaceId } = vi.hoisted(() => ({
  mockNavigate: vi.fn<(input: unknown) => Promise<void>>(),
  mockSetActiveWorkspaceId: vi.fn<(workspaceId: string | null) => void>(),
}));

let mockActiveWorkspaceId: string | null = "ws_alpha";
let mockSelectedWorkspaceId: string | null = "ws_alpha";
let mockChildParams: {
  workspaceId?: string;
  channel?: string;
  threadId?: string;
  directId?: string;
} = {
  workspaceId: "ws_alpha",
  channel: "copy",
};
let mockChildPathname = "/network/ws_alpha/copy/threads";

vi.mock("@tanstack/react-router", () => ({
  useChildMatches: () => [{ pathname: mockChildPathname }],
  useNavigate: () => mockNavigate,
  useParams: () => mockChildParams,
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspaceId: mockActiveWorkspaceId,
    selectedWorkspaceId: mockSelectedWorkspaceId,
    setActiveWorkspaceId: mockSetActiveWorkspaceId,
  }),
}));

vi.mock("../use-last-read", () => ({
  useLastRead: () => ({
    lastReadAt: vi.fn(() => null),
  }),
}));

vi.mock("../use-network-page", () => ({
  useNetworkPage: () => ({
    channels: [
      {
        channel: "copy",
        last_activity_at: "2026-05-14T02:00:00Z",
      },
    ],
    firstVisibleChannel: { channel: "copy" },
    recents: [],
  }),
}));

import { useNetworkRouteShell } from "../use-network-route-shell";

describe("useNetworkRouteShell", () => {
  beforeEach(() => {
    mockActiveWorkspaceId = "ws_alpha";
    mockSelectedWorkspaceId = "ws_alpha";
    mockChildParams = {
      workspaceId: "ws_alpha",
      channel: "copy",
    };
    mockChildPathname = "/network/ws_alpha/copy/threads";
    mockNavigate.mockReset();
    mockSetActiveWorkspaceId.mockReset();
    mockNavigate.mockResolvedValue(undefined);
  });

  it("uses the route workspace when opening a deep Network URL directly", async () => {
    mockActiveWorkspaceId = "ws_beta";
    mockSelectedWorkspaceId = null;
    mockChildParams = {
      workspaceId: "ws_alpha",
      channel: "copy",
    };

    renderHook(() => useNetworkRouteShell());

    await waitFor(() => {
      expect(mockSetActiveWorkspaceId).toHaveBeenCalledWith("ws_alpha");
    });
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("does not let a stale detail route overwrite an explicit workspace switch", async () => {
    const { rerender } = renderHook(() => useNetworkRouteShell());

    mockActiveWorkspaceId = "ws_beta";
    mockSelectedWorkspaceId = "ws_beta";
    rerender();

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith({ to: "/network" });
    });
    expect(mockSetActiveWorkspaceId).not.toHaveBeenCalledWith("ws_alpha");
  });
});
