import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TopbarSlotProvider, useTopbarSlotValue, type TopbarSlotValue } from "@agh/ui";
import type {
  InspectorMemoryState,
  SessionLedgerResponse,
  SessionPayload,
} from "@/systems/session";
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
  mockNavigate,
  mockUseSession,
  mockUseSessionVaultSecrets,
  mockUseSessionLedger,
  mockSessionInspector,
  mockResume,
  mockStop,
  mockClear,
  mockDelete,
} = vi.hoisted(() => ({
  mockNavigate: vi.fn(),
  mockUseSession: vi.fn(),
  mockUseSessionVaultSecrets: vi.fn<(sessionId: string) => SessionVaultQueryState>(() => ({
    data: [],
    isLoading: false,
    error: null,
  })),
  mockUseSessionLedger: vi.fn<
    (sessionId: string, options?: SessionLedgerHookOptions) => SessionLedgerQueryState
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
}));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: (_path: string) => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
    useParams: () => ({ name: "claude-agent", id: "sess_123" }),
  }),
  useNavigate: () => mockNavigate,
}));

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock("@/components/assistant-ui/session-thread", () => ({
  SessionThread: ({ sessionId }: { sessionId: string }) => (
    <div data-testid={`session-thread-${sessionId}`}>thread</div>
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
  useSessionLedger: (id: string, options?: SessionLedgerHookOptions) =>
    mockUseSessionLedger(id, options),
}));

vi.mock("@/systems/vault", () => ({
  useSessionVaultSecrets: (sessionId: string) => mockUseSessionVaultSecrets(sessionId),
}));

vi.mock("@/systems/session/hooks/use-session-actions", () => ({
  useResumeSession: () => mockResume,
  useStopSession: () => mockStop,
  useClearSessionConversation: () => mockClear,
  useDeleteSession: () => mockDelete,
}));

vi.mock("@/systems/session/adapters/session-api", () => ({
  cancelSessionPrompt: vi.fn(),
}));

vi.mock("@assistant-ui/react", () => ({
  useAui: () => ({ thread: () => ({ reset: vi.fn() }) }),
  useAuiState: <T,>(
    selector: (state: { thread: { messages: unknown[]; isRunning: boolean } }) => T
  ) => selector({ thread: { messages: [], isRunning: false } }),
}));

import { SessionPage } from "../agents.$name.sessions.$id";

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
  const utils = render(
    <TopbarSlotProvider>
      <SessionPage />
      <TopbarSlotProbe slotRef={slotRef} />
    </TopbarSlotProvider>
  );
  return { ...utils, slotRef };
}

function makeSession(overrides: Partial<SessionPayload> = {}): SessionPayload {
  return {
    id: "sess_123",
    agent_name: "claude-agent",
    provider: "codex",
    workspace_id: "ws_alpha",
    workspace_path: "/workspace/alpha",
    state: "stopped",
    name: "Old runtime",
    created_at: "2026-04-20T10:00:00Z",
    updated_at: "2026-04-20T11:00:00Z",
    ...overrides,
  };
}

describe("Nested agent session route — Topbar slot migration", () => {
  beforeEach(() => {
    mockNavigate.mockReset();
    mockResume.mutate.mockReset();
    mockResume.isPending = false;
    mockStop.mutate.mockReset();
    mockClear.mutate.mockReset();
    mockDelete.mutate.mockReset();
    mockUseSession.mockReset();
    mockUseSessionVaultSecrets.mockReset();
    mockUseSessionVaultSecrets.mockReturnValue({ data: [], isLoading: false, error: null });
    mockUseSessionLedger.mockReset();
    mockUseSessionLedger.mockReturnValue({ data: undefined, isLoading: false, error: null });
    mockSessionInspector.mockClear();
    mockUseSession.mockReturnValue({
      data: makeSession(),
      isLoading: false,
      error: null,
    });
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

  it("Should render the agent state + provider as bare mono identifiers in the Topbar meta slot", () => {
    renderSessionPage();
    const meta = screen.getByTestId("session-topbar-meta");
    expect(meta).toBeInTheDocument();
    const state = screen.getByTestId("session-topbar-state");
    expect(state).toHaveTextContent("stopped");
    expect(state.className).toContain("font-mono");
    expect(state.className).toContain("text-(--faint)");
    const provider = screen.getByTestId("session-topbar-provider");
    expect(provider).toHaveTextContent("codex");
    expect(provider.className).toContain("font-mono");
  });

  it("Should expose delete/stop/resume controls inside the Topbar actions slot for stopped sessions", () => {
    renderSessionPage();
    expect(screen.getByTestId("session-topbar-actions")).toBeInTheDocument();
    expect(screen.getByTestId("delete-button")).toBeInTheDocument();
    expect(screen.getByTestId("resume-button")).toBeInTheDocument();
    expect(screen.queryByTestId("stop-button")).not.toBeInTheDocument();
  });

  it("Should expose the stop control inside the Topbar actions slot for active sessions", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "active" }),
      isLoading: false,
      error: null,
    });
    renderSessionPage();
    expect(screen.getByTestId("stop-button")).toBeInTheDocument();
    expect(screen.queryByTestId("resume-button")).not.toBeInTheDocument();
  });

  it("Should flip the agent-status-dot to warning+pulse for starting sessions", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "starting" }),
      isLoading: false,
      error: null,
    });
    renderSessionPage();
    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.getAttribute("data-tone")).toBe("warning");
    expect(dot.getAttribute("data-pulse")).toBe("true");
  });
});

describe("Nested agent session route — resume failure UX", () => {
  beforeEach(() => {
    mockNavigate.mockReset();
    mockResume.mutate.mockReset();
    mockResume.isPending = false;
    mockStop.mutate.mockReset();
    mockClear.mutate.mockReset();
    mockDelete.mutate.mockReset();
    mockUseSession.mockReset();
    mockUseSessionVaultSecrets.mockReset();
    mockUseSessionVaultSecrets.mockReturnValue({ data: [], isLoading: false, error: null });
    mockUseSessionLedger.mockReset();
    mockUseSessionLedger.mockReturnValue({ data: undefined, isLoading: false, error: null });
    mockSessionInspector.mockClear();
    mockUseSession.mockReturnValue({
      data: makeSession(),
      isLoading: false,
      error: null,
    });
  });

  it("renders a dedicated inline resume-failure state when resume rejects with a provider-validation error", () => {
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
    mockUseSessionLedger.mockReturnValue({ data: ledger, isLoading: false, error: null });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", { enabled: true });
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

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", { enabled: false });
  });

  it("disables the ledger query while the session is starting", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "starting" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", { enabled: false });
  });

  it("disables the ledger query while the session is stopping", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "stopping" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", { enabled: false });
  });

  it("enables the ledger query once the session has stopped", () => {
    mockUseSession.mockReturnValue({
      data: makeSession({ state: "stopped" }),
      isLoading: false,
      error: null,
    });

    renderSessionPage();

    expect(mockUseSessionLedger).toHaveBeenCalledWith("sess_123", { enabled: true });
  });
});
