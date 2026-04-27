import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SessionPayload } from "@/systems/session/types";

const {
  mockNavigate,
  mockUseSession,
  mockUseWorkspaces,
  mockResume,
  mockStop,
  mockClear,
  mockDelete,
} = vi.hoisted(() => ({
  mockNavigate: vi.fn(),
  mockUseSession: vi.fn(),
  mockUseWorkspaces: vi.fn(() => ({ data: [] })),
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
  SessionInspector: () => <div data-testid="session-inspector">inspector</div>,
}));

vi.mock("@/systems/session/hooks/use-sessions", () => ({
  useSession: (id: string) => mockUseSession(id),
}));

vi.mock("@/systems/workspace", () => ({
  useWorkspaces: () => mockUseWorkspaces(),
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

import { Route } from "./agents.$name.sessions.$id";

const SessionPage = (Route as unknown as { component: () => ReactNode }).component;

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

describe("Nested agent session route — resume failure UX", () => {
  beforeEach(() => {
    mockNavigate.mockReset();
    mockResume.mutate.mockReset();
    mockResume.isPending = false;
    mockStop.mutate.mockReset();
    mockClear.mutate.mockReset();
    mockDelete.mutate.mockReset();
    mockUseSession.mockReset();
    mockUseWorkspaces.mockReset();
    mockUseWorkspaces.mockReturnValue({ data: [] });
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

    render(<SessionPage />);

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

    render(<SessionPage />);

    fireEvent.click(screen.getByTestId("resume-button"));
    expect(screen.getByTestId("session-resume-failure")).toBeInTheDocument();

    act(() => {
      fireEvent.click(screen.getByTestId("session-resume-failure-dismiss"));
    });

    expect(screen.queryByTestId("session-resume-failure")).not.toBeInTheDocument();
  });

  it("renders the effective provider badge in the chat header", () => {
    render(<SessionPage />);
    expect(screen.getByTestId("session-provider-badge")).toHaveTextContent("codex");
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

    render(<SessionPage />);

    fireEvent.click(screen.getByTestId("delete-button"));
    fireEvent.click(screen.getByTestId("delete-dialog-confirm"));

    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/agents/$name",
      params: { name: "codex-agent" },
    });
  });
});
