import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TopbarSlotProvider, useTopbarSlotValue, type TopbarSlotValue } from "@agh/ui";
import {
  sessionKeys,
  type InspectorMemoryState,
  type SessionLedgerResponse,
  type SessionPayload,
} from "@/systems/session";
import { sessionTranscriptFixture } from "@/systems/session/mocks/fixtures";
import { useActiveWorkspaceStore, type WorkspacePayload } from "@/systems/workspace";
import type { VaultSecret } from "@/systems/vault";

type SessionVaultQueryState = {
  data: VaultSecret[];
  isLoading: boolean;
  error: Error | null;
};

type SessionLedgerQueryState = {
  data: SessionLedgerResponse | undefined;
  isLoading: boolean;
  error: Error | null;
};

type SessionLedgerHookOptions = { enabled?: boolean } | undefined;

type SessionInspectorPropsForTest = {
  sessionId?: string;
  memory?: InspectorMemoryState;
  vaultSecrets?: VaultSecret[];
  vaultIsLoading?: boolean;
  vaultError?: Error | null;
};

const {
  mockRedirect,
  mockNavigate,
  mockUseSession,
  mockUseSessionVaultSecrets,
  mockUseSessionLedger,
  mockSessionInspector,
  mockResume,
  mockStop,
  mockClear,
  mockDelete,
  mockQueuePrompt,
  mockInterruptPrompt,
  mockSteerPrompt,
  mockCancelQueuedPrompt,
  mockRouteTranscriptMessages,
  mockRouteLoaderData,
} = vi.hoisted(() => ({
  mockRedirect: vi.fn((args: unknown) => ({ __redirect: true, ...(args as object) })),
  mockNavigate: vi.fn(),
  mockUseSession: vi.fn(),
  mockUseSessionVaultSecrets: vi.fn<(sessionId: string) => SessionVaultQueryState>(() => ({
    data: [],
    isLoading: false,
    error: null,
  })),
  mockUseSessionLedger: vi.fn<
    (
      sessionId: string,
      workspaceId?: string | null,
      options?: SessionLedgerHookOptions
    ) => SessionLedgerQueryState
  >(() => ({
    data: undefined,
    isLoading: false,
    error: null,
  })),
  mockSessionInspector: vi.fn<(props: SessionInspectorPropsForTest) => ReactNode>(() => (
    <div data-testid="session-inspector">inspector</div>
  )),
  mockResume: {
    mutate: vi.fn<(id: string, opts?: { onError?: (error: unknown) => void }) => void>(),
    isPending: false as boolean,
  },
  mockStop: { mutate: vi.fn(), isPending: false },
  mockClear: {
    mutate: vi.fn(),
    isPending: false,
  },
  mockDelete: {
    mutate: vi.fn(),
    isPending: false,
  },
  mockQueuePrompt: {
    mutateAsync: vi.fn(),
    isPending: false,
  },
  mockInterruptPrompt: {
    mutateAsync: vi.fn(),
    isPending: false,
  },
  mockSteerPrompt: {
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  },
  mockCancelQueuedPrompt: {
    mutate: vi.fn(),
    isPending: false,
  },
  mockRouteTranscriptMessages: [] as Array<{
    parts?: Array<{ type?: string; text?: string }>;
  }>,
  mockRouteLoaderData: { workspaceId: "ws_alpha" as string | null },
}));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute:
    (_path: string) =>
    (opts: {
      beforeLoad?: (args: unknown) => unknown;
      loader?: (args: unknown) => unknown;
      component: () => ReactNode;
    }) => ({
      ...opts,
      useParams: () => ({ name: "claude-agent", id: "sess_123" }),
      useLoaderData: () => mockRouteLoaderData,
    }),
  useNavigate: () => mockNavigate,
  redirect: mockRedirect,
}));

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    warning: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock("@/components/assistant-ui/session-thread", () => ({
  SessionThread: ({
    sessionId,
    canPrompt,
    allowBusyInput,
    isSessionRunning,
  }: {
    sessionId: string;
    canPrompt?: boolean;
    allowBusyInput?: boolean;
    isSessionRunning?: boolean;
  }) => (
    <div
      data-testid={`session-thread-${sessionId}`}
      data-can-prompt={String(canPrompt)}
      data-allow-busy-input={String(allowBusyInput)}
      data-session-running={String(isSessionRunning)}
    >
      {mockRouteTranscriptMessages.length > 0
        ? mockRouteTranscriptMessages
            .flatMap(message => message.parts ?? [])
            .filter(part => part.type === "text" && typeof part.text === "string")
            .map(part => <p key={part.text}>{part.text}</p>)
        : "Start a conversation."}
    </div>
  ),
}));

vi.mock("@/systems/session/components/session-chat-runtime-provider", () => ({
  SessionChatRuntimeProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/systems/session/components/session-inspector", () => ({
  SessionInspector: mockSessionInspector,
}));

vi.mock("@/systems/session/hooks/use-sessions", () => ({
  useSession: (id: string) => mockUseSession(id),
  useSessionById: (id: string, workspaceId?: string | null) => mockUseSession(id, workspaceId),
  useSessionLedger: (id: string, workspaceId?: string | null, options?: SessionLedgerHookOptions) =>
    mockUseSessionLedger(id, workspaceId, options),
  useSessionUsage: () => ({ data: undefined }),
}));

vi.mock("@/systems/workspace/adapters/workspace-api", () => ({
  fetchWorkspace: vi.fn(),
  fetchWorkspaces: vi.fn(),
  resolveWorkspace: vi.fn(),
}));

vi.mock("@/systems/vault", () => ({
  useSessionVaultSecrets: (sessionId: string) => mockUseSessionVaultSecrets(sessionId),
}));

vi.mock("@/systems/session/hooks/use-session-actions", () => ({
  useResumeSession: () => mockResume,
  useStopSession: () => mockStop,
  useClearSessionConversation: () => mockClear,
  useDeleteSession: () => mockDelete,
  useQueueSessionPrompt: () => mockQueuePrompt,
  useInterruptSessionPrompt: () => mockInterruptPrompt,
  useSteerSessionPrompt: () => mockSteerPrompt,
  useCancelQueuedSessionPrompt: () => mockCancelQueuedPrompt,
}));

vi.mock("@/systems/session/adapters/session-api", () => ({
  approveSession: vi.fn(),
  cancelQueuedSessionPrompt: vi.fn(),
  cancelSessionPrompt: vi.fn(),
  createSession: vi.fn(),
  deleteSession: vi.fn(),
  fetchSession: vi.fn(),
  fetchSessionById: vi.fn(),
  fetchSessionEvents: vi.fn(),
  fetchSessionHistory: vi.fn(),
  fetchSessionLedger: vi.fn(),
  fetchSessionRecap: vi.fn(),
  fetchSessionUsage: vi.fn(),
  fetchSessions: vi.fn(),
  fetchSessionTranscript: vi.fn(),
  interruptSessionPrompt: vi.fn(),
  repairSession: vi.fn(),
  resumeSession: vi.fn(),
  sendSessionPrompt: vi.fn(),
  steerSessionPrompt: vi.fn(),
  stopSession: vi.fn(),
  SessionApiError: class SessionApiError extends Error {
    constructor(
      message: string,
      public readonly status: number,
      public readonly sessionId?: string
    ) {
      super(message);
      this.name = "SessionApiError";
    }
  },
  SessionLedgerUnavailableError: class SessionLedgerUnavailableError extends Error {},
  SessionNotFoundError: class SessionNotFoundError extends Error {
    constructor(public readonly sessionId: string) {
      super(`Session not found: ${sessionId}`);
      this.name = "SessionNotFoundError";
    }
  },
}));

vi.mock("@assistant-ui/react", () => ({
  useAui: () => ({ thread: () => ({ reset: vi.fn() }) }),
  useAuiState: <T,>(
    selector: (state: { thread: { messages: unknown[]; isRunning: boolean } }) => T
  ) => selector({ thread: { messages: [], isRunning: false } }),
}));

import { prefetchAgentSessionRoute, SessionPage } from "../agents.$name.sessions.$id";
import { redirectSessionPermalinkRoute, resolveSessionPermalink } from "../session.$id";
import {
  fetchSession,
  fetchSessionById,
  fetchSessions,
  fetchSessionTranscript,
  SessionApiError,
} from "@/systems/session/adapters/session-api";
import { fetchWorkspaces } from "@/systems/workspace/adapters/workspace-api";
import { toast } from "sonner";

function TopbarSlotProbe({ slotRef }: { slotRef: { current: TopbarSlotValue | null } }) {
  const slot = useTopbarSlotValue();
  slotRef.current = slot;
  return (
    <div data-testid="topbar-probe">
      <span data-testid="topbar-probe-title">
        {typeof slot?.title === "string" ? slot.title : ""}
      </span>
      <div data-testid="topbar-probe-meta">{slot?.meta ?? null}</div>
      <div data-testid="topbar-probe-actions">{slot?.actions ?? null}</div>
    </div>
  );
}

function renderSessionPage() {
  const slotRef: { current: TopbarSlotValue | null } = { current: null };
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <TopbarSlotProvider>
        <SessionPage />
        <TopbarSlotProbe slotRef={slotRef} />
      </TopbarSlotProvider>
    </QueryClientProvider>
  );
  return { ...utils, slotRef };
}

function makeWorkspace(overrides: Partial<WorkspacePayload> = {}): WorkspacePayload {
  return {
    id: "ws_alpha",
    name: "alpha",
    root_dir: "/workspace/alpha",
    add_dirs: [],
    created_at: "2026-04-20T10:00:00Z",
    updated_at: "2026-04-20T10:00:00Z",
    ...overrides,
  };
}

function makeSession(overrides: Partial<SessionPayload> = {}): SessionPayload {
  return {
    id: "sess_123",
    agent_name: "claude-agent",
    provider: "codex",
    workspace_id: "ws_alpha",
    workspace_path: "/workspace/alpha",
    state: "stopped",
    badge: "stopped",
    attachable: false,
    name: "Old runtime",
    created_at: "2026-04-20T10:00:00Z",
    updated_at: "2026-04-20T11:00:00Z",
    ...overrides,
  };
}

describe("Nested agent session route — Topbar slot migration", () => {
  beforeEach(() => {
    mockRedirect.mockClear();
    mockNavigate.mockReset();
    mockResume.mutate.mockReset();
    mockResume.isPending = false;
    mockStop.mutate.mockReset();
    mockClear.mutate.mockReset();
    mockDelete.mutate.mockReset();
    mockQueuePrompt.mutateAsync.mockReset();
    mockQueuePrompt.isPending = false;
    mockInterruptPrompt.mutateAsync.mockReset();
    mockInterruptPrompt.isPending = false;
    mockSteerPrompt.mutateAsync.mockReset();
    mockSteerPrompt.isPending = false;
    mockRouteTranscriptMessages.length = 0;
    mockRouteLoaderData.workspaceId = "ws_alpha";
    mockUseSession.mockReset();
    mockUseSessionVaultSecrets.mockReset();
    mockUseSessionVaultSecrets.mockReturnValue({ data: [], isLoading: false, error: null });
    mockUseSessionLedger.mockReset();
    mockUseSessionLedger.mockReturnValue({ data: undefined, isLoading: false, error: null });
    mockSessionInspector.mockClear();
    vi.mocked(fetchSession).mockReset();
    vi.mocked(fetchSessionById).mockReset();
    vi.mocked(fetchSessionTranscript).mockReset();
    vi.mocked(fetchSessions).mockReset();
    vi.mocked(fetchWorkspaces).mockReset();
    vi.mocked(toast.warning).mockClear();
    vi.mocked(fetchWorkspaces).mockResolvedValue([makeWorkspace()]);
    useActiveWorkspaceStore.setState({ selectedWorkspaceId: null });
    mockUseSession.mockReturnValue({
      data: makeSession(),
      isLoading: false,
      error: null,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("Should never render the legacy <ChatHeader>", () => {
    renderSessionPage();
    expect(screen.queryByTestId("chat-header")).not.toBeInTheDocument();
    expect(screen.queryByTestId("chat-breadcrumb")).not.toBeInTheDocument();
  });

  it("Should push the agent name into the Topbar title slot", () => {
    const { slotRef } = renderSessionPage();
    expect(slotRef.current?.title).toBe("claude-agent");
  });

  it("Should render the daemon badge + provider as bare mono identifiers in the Topbar meta slot", () => {
    renderSessionPage();
    const meta = screen.getByTestId("session-topbar-meta");
    expect(meta).toBeInTheDocument();
    const badge = screen.getByTestId("session-topbar-badge");
    expect(badge).toHaveTextContent("stopped");
    const provider = screen.getByTestId("session-topbar-provider");
    expect(provider).toHaveTextContent("codex");
  });

  it("Should expose delete without attach controls for stopped sessions", () => {
    renderSessionPage();
    expect(screen.getByTestId("session-topbar-actions")).toBeInTheDocument();
    expect(screen.getByTestId("delete-button")).toBeInTheDocument();
    expect(screen.queryByTestId("resume-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("stop-button")).not.toBeInTheDocument();
  });

  it("Should expose stop and attach controls for attachable active sessions", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "active", badge: "idle", attachable: true, type: "user" }),
      isLoading: false,
      error: null,
    });
    renderSessionPage();
    expect(screen.getByTestId("stop-button")).toBeInTheDocument();
    expect(screen.getByTestId("resume-button")).toHaveAccessibleName("Attach session");
  });

  it("Should expose view plus Stop only for attachable non-user running sessions", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "active", badge: "running", attachable: true, type: "spawned" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    const thread = screen.getByTestId("session-thread-sess_123");
    expect(thread).toHaveAttribute("data-can-prompt", "false");
    expect(thread).toHaveAttribute("data-allow-busy-input", "false");
    expect(thread).toHaveAttribute("data-session-running", "true");
    expect(screen.getByTestId("stop-button")).toBeInTheDocument();
    expect(screen.queryByTestId("resume-button")).not.toBeInTheDocument();
  });

  it("Should render warm transcript rows on session detail route remount without empty-state flash", () => {
    mockRouteTranscriptMessages.push(...sessionTranscriptFixture.slice(0, 2));

    const firstRender = renderSessionPage();
    expect(
      screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
    ).toBeInTheDocument();
    expect(screen.getByText(/Launch readiness snapshot/)).toBeInTheDocument();

    firstRender.unmount();
    renderSessionPage();

    expect(screen.queryByText("Start a conversation.")).not.toBeInTheDocument();
    expect(
      screen.getAllByText("Summarize the launch blockers before the 18:30 UTC cutover.")
    ).toHaveLength(1);
    expect(screen.getAllByText(/Launch readiness snapshot/)).toHaveLength(1);
  });

  it("Should keep warm loader data past the old 5-minute gcTime window and render transcript rows immediately", async () => {
    vi.useFakeTimers();
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const session = makeSession({ state: "active", badge: "running", attachable: true });
    const messages = sessionTranscriptFixture.slice(0, 2);
    useActiveWorkspaceStore.setState({ selectedWorkspaceId: "ws_beta" });
    vi.mocked(fetchSessionById).mockResolvedValue(session);
    vi.mocked(fetchSession).mockResolvedValue(session);
    vi.mocked(fetchSessionTranscript).mockResolvedValue(messages);
    vi.mocked(fetchSessions).mockResolvedValue([]);
    queryClient.setQueryData(sessionKeys.list("ws_beta"), [
      makeSession({ id: session.id, workspace_id: "ws_beta", workspace_path: "/workspace/beta" }),
    ]);

    const loaderData = await prefetchAgentSessionRoute({
      queryClient,
      sessionId: session.id,
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5 * 60 * 1_000 + 1);
    });

    const cachedSession = queryClient.getQueryData<SessionPayload>(
      sessionKeys.detail("ws_alpha", session.id)
    );
    const cachedTranscript = queryClient.getQueryData<typeof messages>(
      sessionKeys.transcript("ws_alpha", session.id)
    );
    expect(cachedSession).toEqual(session);
    expect(cachedTranscript).toEqual(messages);

    mockRouteLoaderData.workspaceId = loaderData.workspaceId;
    mockUseSession.mockReturnValue({
      data: cachedSession,
      isLoading: false,
      error: null,
    });
    mockRouteTranscriptMessages.push(...(cachedTranscript ?? []));

    renderSessionPage();

    expect(screen.queryByText("Start a conversation.")).not.toBeInTheDocument();
    expect(
      screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
    ).toBeInTheDocument();
    expect(screen.getByText(/Launch readiness snapshot/)).toBeInTheDocument();
    expect(fetchSessionById).toHaveBeenCalledWith(session.id, expect.any(AbortSignal));
    expect(fetchSession).toHaveBeenCalledWith("ws_alpha", session.id, expect.any(AbortSignal));
    expect(fetchSessions).not.toHaveBeenCalled();
  });

  it("Should resolve the canonical session route by session id without active-workspace fallback", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const session = makeSession({ state: "active", badge: "running", attachable: true });
    const messages = sessionTranscriptFixture.slice(0, 2);
    useActiveWorkspaceStore.setState({ selectedWorkspaceId: "ws_beta" });
    vi.mocked(fetchWorkspaces).mockResolvedValue([
      makeWorkspace({ id: "ws_beta", name: "beta", root_dir: "/workspace/beta" }),
    ]);
    vi.mocked(fetchSessionById).mockResolvedValue(session);
    vi.mocked(fetchSession).mockResolvedValue(session);
    vi.mocked(fetchSessionTranscript).mockResolvedValue(messages);
    vi.mocked(fetchSessions).mockResolvedValue([]);

    const loaderData = await prefetchAgentSessionRoute({
      queryClient,
      sessionId: session.id,
    });
    expect(fetchWorkspaces).not.toHaveBeenCalled();
    mockRouteLoaderData.workspaceId = loaderData.workspaceId;
    mockUseSession.mockReturnValue({
      data: session,
      isLoading: false,
      error: null,
    });
    mockRouteTranscriptMessages.push(...messages);

    renderSessionPage();

    expect(screen.queryByText("Start a conversation.")).not.toBeInTheDocument();
    expect(
      screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
    ).toBeInTheDocument();
    expect(mockUseSession).toHaveBeenCalledWith(session.id, "ws_alpha");
    expect(fetchSessionById).toHaveBeenCalledWith(session.id, expect.any(AbortSignal));
    expect(fetchSessions).not.toHaveBeenCalled();
    expect(fetchSession).toHaveBeenCalledWith("ws_alpha", session.id, expect.any(AbortSignal));
    expect(fetchSessionTranscript).toHaveBeenCalledWith(
      "ws_alpha",
      session.id,
      expect.any(AbortSignal)
    );
    expect(fetchSessionById).toHaveBeenCalledTimes(1);
    expect(fetchSessionTranscript).toHaveBeenCalledTimes(1);
  });

  it("Should resolve /session/$id by id once and seed the canonical route cache", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const session = makeSession({ state: "active", badge: "running", attachable: true });
    const messages = sessionTranscriptFixture.slice(0, 2);
    vi.mocked(fetchSessionById).mockResolvedValue(session);
    vi.mocked(fetchSessionTranscript).mockResolvedValue(messages);
    vi.mocked(fetchSession).mockResolvedValue(session);

    const resolved = await resolveSessionPermalink({
      queryClient,
      sessionId: session.id,
    });

    expect(resolved).toEqual(session);
    expect(fetchSessionById).toHaveBeenCalledTimes(1);
    expect(fetchSessionById).toHaveBeenCalledWith(session.id, expect.any(AbortSignal));
    expect(fetchSessions).not.toHaveBeenCalled();
    expect(fetchSession).not.toHaveBeenCalled();
    expect(fetchSessionTranscript).toHaveBeenCalledWith(
      "ws_alpha",
      session.id,
      expect.any(AbortSignal)
    );
    expect(queryClient.getQueryData(sessionKeys.detail("ws_alpha", session.id))).toEqual(session);
    expect(queryClient.getQueryData(sessionKeys.transcript("ws_alpha", session.id))).toEqual(
      messages
    );
  });

  it("Should resolve /session/$id when supplementary prefetches fail", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const session = makeSession({ state: "active", badge: "running", attachable: true });
    vi.mocked(fetchSessionById).mockResolvedValue(session);
    vi.mocked(fetchSessionTranscript).mockRejectedValue(new Error("transcript unavailable"));
    vi.mocked(fetchSession).mockRejectedValue(new Error("detail unavailable"));

    const resolved = await resolveSessionPermalink({
      queryClient,
      sessionId: session.id,
    });

    expect(resolved).toEqual(session);
    expect(queryClient.getQueryData(sessionKeys.detail("ws_alpha", session.id))).toEqual(session);
  });

  it("Should rethrow non-not-found lookup failures for the canonical session route", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const error = new SessionApiError("daemon unavailable", 503, "sess_123");
    vi.mocked(fetchSessionById).mockRejectedValue(error);

    await expect(
      prefetchAgentSessionRoute({
        queryClient,
        sessionId: "sess_123",
      })
    ).rejects.toBe(error);
  });

  it("Should redirect /session/$id from beforeLoad to the canonical session route", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const session = makeSession({ state: "active", badge: "running", attachable: true });
    vi.mocked(fetchSessionById).mockResolvedValue(session);
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);

    await expect(
      redirectSessionPermalinkRoute({
        context: { queryClient },
        params: { id: session.id },
      })
    ).rejects.toMatchObject({
      __redirect: true,
      to: "/agents/$name/sessions/$id",
      params: { name: "claude-agent", id: session.id },
      replace: true,
    });

    expect(mockRedirect).toHaveBeenCalledWith({
      to: "/agents/$name/sessions/$id",
      params: { name: "claude-agent", id: session.id },
      replace: true,
    });
  });

  it("Should flip the agent-status-dot to success+pulse for running badges", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "starting", badge: "running" }),
      isLoading: false,
      error: null,
    });
    renderSessionPage();
    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.getAttribute("data-tone")).toBe("success");
    expect(dot.getAttribute("data-pulse")).toBe("true");
  });
});

describe("Nested agent session route — attach failure UX", () => {
  beforeEach(() => {
    mockRedirect.mockClear();
    mockNavigate.mockReset();
    mockResume.mutate.mockReset();
    mockResume.isPending = false;
    mockStop.mutate.mockReset();
    mockClear.mutate.mockReset();
    mockDelete.mutate.mockReset();
    mockQueuePrompt.mutateAsync.mockReset();
    mockQueuePrompt.isPending = false;
    mockInterruptPrompt.mutateAsync.mockReset();
    mockInterruptPrompt.isPending = false;
    mockSteerPrompt.mutateAsync.mockReset();
    mockSteerPrompt.isPending = false;
    mockRouteTranscriptMessages.length = 0;
    mockRouteLoaderData.workspaceId = "ws_alpha";
    mockUseSession.mockReset();
    mockUseSessionVaultSecrets.mockReset();
    mockUseSessionVaultSecrets.mockReturnValue({ data: [], isLoading: false, error: null });
    mockUseSessionLedger.mockReset();
    mockUseSessionLedger.mockReturnValue({ data: undefined, isLoading: false, error: null });
    mockSessionInspector.mockClear();
    vi.mocked(fetchSession).mockReset();
    vi.mocked(fetchSessionById).mockReset();
    vi.mocked(fetchSessionTranscript).mockReset();
    vi.mocked(fetchSessions).mockReset();
    vi.mocked(fetchWorkspaces).mockReset();
    vi.mocked(toast.warning).mockClear();
    vi.mocked(fetchWorkspaces).mockResolvedValue([makeWorkspace()]);
    useActiveWorkspaceStore.setState({ selectedWorkspaceId: null });
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "active", badge: "idle", attachable: true }),
      isLoading: false,
      error: null,
    });
  });

  it("renders a dedicated inline attach-failure state when attach rejects with a provider-validation error", () => {
    mockResume.mutate.mockImplementation((_id, opts) => {
      opts?.onError?.(
        new Error(
          `session: validate resume infrastructure for "sess_123": session: validate agent "claude-agent" with provider "codex" for session "sess_123": workspace: agent not available`
        )
      );
    });

    renderSessionPage();

    fireEvent.click(screen.getByTestId("resume-button"));

    const failure = screen.getByTestId("session-resume-failure");
    expect(failure).toBeInTheDocument();
    expect(screen.getByTestId("session-resume-failure-title")).toHaveTextContent(
      "Attach failed: provider no longer available"
    );
    expect(screen.getByTestId("session-resume-failure-provider")).toHaveTextContent("codex");
    expect(screen.getByTestId("session-resume-failure-meta")).toHaveTextContent("sess_123");
    expect(screen.getByTestId("session-resume-failure-meta")).toHaveTextContent("claude-agent");
  });

  it("dismisses the failure panel via its dismiss action", () => {
    mockResume.mutate.mockImplementation((_id, opts) => {
      opts?.onError?.(
        new Error(
          `session: validate resume infrastructure for "sess_123": session: validate agent "claude-agent" with provider "codex" for session "sess_123": workspace: agent not available`
        )
      );
    });

    renderSessionPage();

    fireEvent.click(screen.getByTestId("resume-button"));
    expect(screen.getByTestId("session-resume-failure")).toBeInTheDocument();

    act(() => {
      fireEvent.click(screen.getByTestId("session-resume-failure-dismiss"));
    });

    expect(screen.queryByTestId("session-resume-failure")).not.toBeInTheDocument();
  });

  it("replaces history when a missing session redirects to the agent page", () => {
    mockUseSession.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("Session not found: sess_123"),
    });

    renderSessionPage();

    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/agents/$name",
      params: { name: "claude-agent" },
      replace: true,
    });
  });

  it("redirects when the loaded session belongs to a different active workspace", async () => {
    vi.mocked(fetchWorkspaces).mockResolvedValue([
      makeWorkspace({ id: "ws_beta", name: "beta", root_dir: "/workspace/beta" }),
    ]);
    mockUseSession.mockReturnValue({
      data: makeSession({ workspace_id: "ws_alpha" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith({
        to: "/agents/$name",
        params: { name: "claude-agent" },
        replace: true,
      });
    });
    expect(toast.warning).toHaveBeenCalledWith(
      'Session "Old runtime" belongs to workspace "ws_alpha".',
      expect.objectContaining({
        id: "session-workspace-mismatch:ws_alpha:sess_123",
        action: expect.objectContaining({ label: "Switch back" }),
      })
    );
  });

  it("navigates to the resolved session agent after delete succeeds", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ agent_name: "codex-agent" }),
      isLoading: false,
      error: null,
    });
    mockDelete.mutate.mockImplementation(
      (_id: string, opts?: { onSuccess?: () => void; onError?: (error: unknown) => void }) => {
        opts?.onSuccess?.();
      }
    );

    renderSessionPage();

    fireEvent.click(screen.getByTestId("delete-button"));
    fireEvent.click(screen.getByTestId("delete-dialog-confirm"));

    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/agents/$name",
      params: { name: "codex-agent" },
    });
  });

  it("passes session-scoped vault metadata into the inspector", () => {
    const vaultSecrets: VaultSecret[] = [
      {
        ref: "vault:sessions/sess_123/github-token",
        namespace: "sessions",
        kind: "token",
        present: true,
        created_at: "2026-05-02T10:00:00Z",
        updated_at: "2026-05-02T10:00:00Z",
      },
    ];
    mockUseSessionVaultSecrets.mockReturnValue({
      data: vaultSecrets,
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    expect(mockUseSessionVaultSecrets).toHaveBeenCalledWith("sess_123");
    const inspectorProps =
      mockSessionInspector.mock.calls[mockSessionInspector.mock.calls.length - 1]?.[0];
    expect(inspectorProps).toMatchObject({
      sessionId: "sess_123",
      vaultSecrets,
      vaultIsLoading: false,
      vaultError: null,
    });
  });

  it("passes the session-scoped ledger query state into the inspector memory prop", () => {
    const ledger: SessionLedgerResponse = {
      meta: {
        version: 1,
        session_id: "sess_123",
        workspace_id: "ws_alpha",
        root_session_id: "sess_root",
        parent_session_id: "sess_parent",
        spawn_depth: 1,
        path: "/sessions/ws_alpha/sess_123/ledger.jsonl",
        checksum: "sha256:abc",
        created_at: "2026-04-20T10:00:00Z",
        stopped_at: "2026-04-20T11:00:00Z",
      },
      events: [
        { sequence: 1, event_type: "session.started", emitted_at: "2026-04-20T10:00:00Z" },
        { sequence: 2, event_type: "memory.recall", emitted_at: "2026-04-20T10:01:00Z" },
      ],
    };
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "stopped", badge: "stopped", attachable: false }),
      isLoading: false,
      error: null,
    });
    mockUseSessionLedger.mockReturnValue({ data: ledger, isLoading: false, error: null });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", "ws_alpha", {
      enabled: true,
    });
    const inspectorProps =
      mockSessionInspector.mock.calls[mockSessionInspector.mock.calls.length - 1]?.[0];
    expect(inspectorProps?.memory).toEqual({
      ledger,
      isLoading: false,
      error: null,
    });
  });

  it("forwards ledger loading state into the inspector memory prop", () => {
    mockUseSessionLedger.mockReturnValue({ data: undefined, isLoading: true, error: null });

    renderSessionPage();

    const inspectorProps =
      mockSessionInspector.mock.calls[mockSessionInspector.mock.calls.length - 1]?.[0];
    expect(inspectorProps?.memory).toEqual({
      ledger: null,
      isLoading: true,
      error: null,
    });
  });

  it("disables the ledger query while the session is still active", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "active" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", "ws_alpha", {
      enabled: false,
    });
  });

  it("disables the ledger query while the session is starting", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "starting" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", "ws_alpha", {
      enabled: false,
    });
  });

  it("disables the ledger query while the session is stopping", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "stopping" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", "ws_alpha", {
      enabled: false,
    });
  });

  it("enables the ledger query once the session has stopped", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "stopped" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", "ws_alpha", {
      enabled: true,
    });
  });
});
