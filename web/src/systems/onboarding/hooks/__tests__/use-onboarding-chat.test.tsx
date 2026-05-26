import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  SessionNotFoundError,
  type CreateSessionParams,
  type SessionPayload,
} from "@/systems/session";

import { useOnboardingDraftStore } from "../../stores/use-onboarding-draft-store";
import { useOnboardingChat } from "../use-onboarding-chat";

const mocks = vi.hoisted(() => ({
  createSession: vi.fn(),
  fetchSession: vi.fn(),
}));

vi.mock("@/systems/session", () => {
  class MockSessionApiError extends Error {
    constructor(
      message: string,
      public readonly status: number,
      public readonly sessionId?: string
    ) {
      super(message);
      this.name = "SessionApiError";
    }
  }

  class MockSessionNotFoundError extends MockSessionApiError {
    constructor(id: string) {
      super(`Session not found: ${id}`, 404, id);
      this.name = "SessionNotFoundError";
    }
  }

  return {
    fetchSession: mocks.fetchSession,
    SessionApiError: MockSessionApiError,
    SessionNotFoundError: MockSessionNotFoundError,
    useCreateSession: () => ({
      isPending: false,
      mutateAsync: mocks.createSession,
    }),
  };
});

const now = "2026-05-25T21:00:00Z";

function sessionPayload(overrides: Partial<SessionPayload> = {}): SessionPayload {
  return {
    id: "sess_active",
    agent_name: "onboarding",
    provider: "codex",
    workspace_id: "ws_alpha",
    workspace_path: "/workspace/alpha",
    state: "active",
    badge: "idle",
    attachable: true,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

function seedDraft(overrides: Partial<ReturnType<typeof useOnboardingDraftStore.getState>> = {}) {
  useOnboardingDraftStore.getState().reset();
  useOnboardingDraftStore.setState({
    workspaces: [{ path: "/workspace/alpha", name: "alpha" }],
    provider: "codex",
    model: "gpt-5.5",
    reasoning: "high",
    ...overrides,
  });
}

async function ensure(
  result: ReturnType<typeof renderHook<ReturnType<typeof useOnboardingChat>, []>>["result"]
) {
  await act(async () => {
    await result.current.ensureSession();
  });
}

describe("useOnboardingChat", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    seedDraft();
    mocks.createSession.mockResolvedValue(
      sessionPayload({
        id: "sess_new",
        workspace_id: "ws_alpha",
      })
    );
  });

  it("clears missing persisted onboarding IDs and creates a fresh session", async () => {
    seedDraft({
      onboardingSessionId: "sess_missing",
      onboardingWorkspaceId: "ws_alpha",
      onboardingKickoffSessionId: "sess_missing",
    });
    mocks.fetchSession.mockRejectedValue(new SessionNotFoundError("sess_missing"));

    const { result } = renderHook(() => useOnboardingChat());

    await ensure(result);

    await waitFor(() => expect(result.current.session?.sessionId).toBe("sess_new"));
    expect(result.current.session).toMatchObject({
      canPrompt: true,
      recoveryMessage: null,
      canRestart: false,
    });
    expect(mocks.fetchSession).toHaveBeenCalledWith("ws_alpha", "sess_missing");
    expect(mocks.createSession).toHaveBeenCalledWith({
      agent_name: "onboarding",
      workspace_path: "/workspace/alpha",
      provider: "codex",
      model: "gpt-5.5",
      reasoning_effort: "high",
    } satisfies CreateSessionParams);
    expect(useOnboardingDraftStore.getState()).toMatchObject({
      onboardingSessionId: "sess_new",
      onboardingWorkspaceId: "ws_alpha",
      onboardingKickoffSessionId: "",
    });
  });

  it("reuses an active persisted onboarding session after backend reconciliation", async () => {
    seedDraft({
      onboardingSessionId: "sess_active",
      onboardingWorkspaceId: "ws_alpha",
      onboardingKickoffSessionId: "",
    });
    mocks.fetchSession.mockResolvedValue(sessionPayload({ id: "sess_active" }));

    const { result } = renderHook(() => useOnboardingChat());

    await ensure(result);

    await waitFor(() => expect(result.current.session?.sessionId).toBe("sess_active"));
    expect(result.current.session).toMatchObject({
      canPrompt: true,
      recoveryMessage: null,
      canRestart: false,
    });
    expect(mocks.fetchSession).toHaveBeenCalledWith("ws_alpha", "sess_active");
    expect(mocks.createSession).not.toHaveBeenCalled();
    expect(result.current.error).toBeNull();
  });

  it("rehydrates stopped failed persisted sessions without creating another session", async () => {
    seedDraft({
      onboardingSessionId: "sess_failed",
      onboardingWorkspaceId: "ws_alpha",
      onboardingKickoffSessionId: "sess_failed",
    });
    mocks.fetchSession.mockResolvedValue(
      sessionPayload({
        id: "sess_failed",
        state: "stopped",
        badge: "failed",
        stop_reason: "agent_crashed",
        stop_detail: "process exited during active prompt",
        failure: {
          kind: "process_exit",
          summary: "peer disconnected before response",
        },
      })
    );

    const { result } = renderHook(() => useOnboardingChat());

    await ensure(result);

    await waitFor(() => expect(result.current.session?.sessionId).toBe("sess_failed"));
    expect(result.current.error).toBeNull();
    expect(result.current.session).toMatchObject({
      canPrompt: false,
      canRestart: true,
      recoveryMessage:
        "This onboarding session stopped before setup finished. The history below is preserved.",
    });
    expect(mocks.createSession).not.toHaveBeenCalled();
    expect(useOnboardingDraftStore.getState()).toMatchObject({
      onboardingSessionId: "sess_failed",
      onboardingWorkspaceId: "ws_alpha",
      onboardingKickoffSessionId: "sess_failed",
    });
  });

  it("retry clears persisted onboarding IDs and starts a new session", async () => {
    seedDraft({
      onboardingSessionId: "sess_failed",
      onboardingWorkspaceId: "ws_alpha",
      onboardingKickoffSessionId: "sess_failed",
    });
    mocks.fetchSession.mockResolvedValue(
      sessionPayload({
        id: "sess_failed",
        state: "stopped",
        badge: "failed",
        failure: {
          kind: "process_exit",
          summary: "peer disconnected before response",
        },
      })
    );

    const { result } = renderHook(() => useOnboardingChat());

    await ensure(result);
    await waitFor(() => expect(result.current.session?.canRestart).toBe(true));

    await act(async () => {
      await result.current.retry();
    });

    await waitFor(() => expect(result.current.session?.sessionId).toBe("sess_new"));
    expect(result.current.session).toMatchObject({
      canPrompt: true,
      recoveryMessage: null,
      canRestart: false,
    });
    expect(mocks.createSession).toHaveBeenCalledTimes(1);
    expect(useOnboardingDraftStore.getState()).toMatchObject({
      onboardingSessionId: "sess_new",
      onboardingWorkspaceId: "ws_alpha",
      onboardingKickoffSessionId: "",
    });
  });
});
