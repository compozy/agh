import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const guardMocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  setActiveWorkspaceId: vi.fn(),
  toastWarning: vi.fn(),
  activeWorkspace: {
    activeWorkspaceId: "ws_beta" as string | null,
    workspaces: [
      { id: "ws_alpha", name: "Alpha workspace" },
      { id: "ws_beta", name: "Beta workspace" },
    ],
  },
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => guardMocks.navigate,
}));

vi.mock("sonner", () => ({
  toast: {
    warning: guardMocks.toastWarning,
  },
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspaceId: guardMocks.activeWorkspace.activeWorkspaceId,
    setActiveWorkspaceId: guardMocks.setActiveWorkspaceId,
    workspaces: guardMocks.activeWorkspace.workspaces,
  }),
}));

import { useSessionWorkspaceGuard } from "../use-session-workspace-guard";

describe("useSessionWorkspaceGuard", () => {
  beforeEach(() => {
    guardMocks.navigate.mockReset();
    guardMocks.setActiveWorkspaceId.mockReset();
    guardMocks.toastWarning.mockReset();
    guardMocks.activeWorkspace.activeWorkspaceId = "ws_beta";
    guardMocks.activeWorkspace.workspaces = [
      { id: "ws_alpha", name: "Alpha workspace" },
      { id: "ws_beta", name: "Beta workspace" },
    ];
  });

  it("Should redirect and show an explanatory notice when the session belongs to another workspace", () => {
    renderHook(() =>
      useSessionWorkspaceGuard({
        agentName: "claude-agent",
        sessionId: "sess_123",
        sessionName: "Launch review",
        sessionWorkspaceId: "ws_alpha",
      })
    );

    expect(guardMocks.navigate).toHaveBeenCalledWith({
      to: "/agents/$name",
      params: { name: "claude-agent" },
      replace: true,
    });
    expect(guardMocks.toastWarning).toHaveBeenCalledWith(
      'Session "Launch review" belongs to workspace "Alpha workspace".',
      expect.objectContaining({
        id: "session-workspace-mismatch:ws_alpha:sess_123",
        action: expect.objectContaining({ label: "Switch back" }),
      })
    );

    const toastOptions = guardMocks.toastWarning.mock.calls[0]?.[1] as {
      action?: { onClick?: () => void };
    };
    act(() => {
      toastOptions.action?.onClick?.();
    });

    expect(guardMocks.setActiveWorkspaceId).toHaveBeenCalledWith("ws_alpha");
    expect(guardMocks.navigate).toHaveBeenLastCalledWith({
      to: "/agents/$name/sessions/$id",
      params: { name: "claude-agent", id: "sess_123" },
      replace: true,
    });
  });

  it("Should not redirect or notify when the active workspace owns the session", () => {
    guardMocks.activeWorkspace.activeWorkspaceId = "ws_alpha";

    renderHook(() =>
      useSessionWorkspaceGuard({
        agentName: "claude-agent",
        sessionId: "sess_123",
        sessionName: "Launch review",
        sessionWorkspaceId: "ws_alpha",
      })
    );

    expect(guardMocks.navigate).not.toHaveBeenCalled();
    expect(guardMocks.toastWarning).not.toHaveBeenCalled();
  });
});
