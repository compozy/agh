import { act, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useSessionStore } from "@/systems/session/hooks/use-session-store";
import type { UIMessage } from "@/systems/session/types";

const mockNavigate = vi.fn();

let routeParams = { id: "sess-1" };
let sessionState: {
  data: {
    id: string;
    name?: string;
    agent_name: string;
    workspace_id: string;
    workspace_path: string;
    state: "starting" | "active" | "stopping" | "stopped";
    created_at: string;
    updated_at: string;
  } | null;
  isLoading: boolean;
  error: Error | null;
};
let transcriptState: {
  transcriptMessages: UIMessage[] | undefined;
  isLoadingTranscript: boolean;
  error: Error | null;
};

function makeSession(id: string) {
  return {
    id,
    agent_name: "coder",
    workspace_id: "ws_alpha",
    workspace_path: "/workspace",
    state: "active" as const,
    created_at: "2026-04-03T12:00:00Z",
    updated_at: "2026-04-03T12:00:00Z",
  };
}

function makeMessage(id: string, content: string): UIMessage {
  return {
    id,
    role: "assistant",
    content,
    timestamp: Date.parse("2026-04-03T12:00:00Z"),
    isStreaming: false,
  };
}

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
    useParams: () => routeParams,
  }),
  useNavigate: () => mockNavigate,
}));

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
  },
}));

vi.mock("@/systems/session/hooks/use-sessions", () => ({
  useSession: () => sessionState,
}));

vi.mock("@/systems/session/hooks/use-session-transcript", () => ({
  useSessionTranscript: () => transcriptState,
}));

vi.mock("@/systems/session/hooks/use-session-chat", () => ({
  useSessionChat: () => ({
    sendMessage: vi.fn(),
    stop: vi.fn(),
    resetLiveState: vi.fn(),
    status: "ready" as const,
    isStoppingPrompt: false,
  }),
}));

vi.mock("@/systems/workspace", () => ({
  useWorkspaces: () => ({
    data: [
      {
        id: "ws_alpha",
        root_dir: "/workspace",
        add_dirs: [],
        name: "alpha",
        created_at: "2026-04-03T12:00:00Z",
        updated_at: "2026-04-03T12:00:00Z",
      },
    ],
  }),
}));

vi.mock("@/systems/session/hooks/use-session-actions", () => ({
  useStopSession: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useResumeSession: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useClearSessionConversation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

vi.mock("@/systems/network", () => ({
  useNetworkChannels: () => ({ data: undefined, isLoading: false, error: null }),
}));

vi.mock("@/systems/session/components/chat-header", () => ({
  ChatHeader: ({ workspaceName, canClear }: { workspaceName?: string; canClear?: boolean }) => (
    <div data-testid="chat-header">{`${workspaceName ?? "no-workspace"}:${String(canClear)}`}</div>
  ),
}));

vi.mock("@/systems/session/components/chat-view", () => ({
  ChatView: ({ messages }: { messages: UIMessage[] }) => (
    <div data-testid="chat-view">
      <span data-testid="message-count">{messages.length}</span>
      <span data-testid="message-content">
        {messages.map(message => message.content).join("|")}
      </span>
    </div>
  ),
}));

vi.mock("@/systems/session/components/message-composer", () => ({
  MessageComposer: ({ inert }: { inert?: boolean }) => (
    <div data-testid="message-composer" data-inert={String(Boolean(inert))} />
  ),
}));

vi.mock("@/systems/session/components/permission-prompt", () => ({
  PermissionPrompt: () => <div data-testid="permission-prompt" />,
}));

import { Route } from "./session.$id";

describe("SessionPage", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const SessionPage = (Route as any).component as () => React.ReactNode;

  beforeEach(() => {
    routeParams = { id: "sess-1" };
    sessionState = {
      data: makeSession("sess-1"),
      isLoading: false,
      error: null,
    };
    transcriptState = {
      transcriptMessages: undefined,
      isLoadingTranscript: false,
      error: null,
    };
    useSessionStore.setState({
      activeSessionId: null,
      historyMessages: [],
      liveMessages: [],
      isStreaming: false,
      awaitingTranscriptSync: false,
      pendingPermission: null,
    });
    mockNavigate.mockReset();
  });

  it("hydrates a late transcript for the active session", () => {
    const { rerender } = render(<SessionPage />);

    expect(screen.getByTestId("chat-header")).toHaveTextContent("alpha:false");
    expect(screen.getByTestId("message-count")).toHaveTextContent("0");

    transcriptState = {
      ...transcriptState,
      transcriptMessages: [makeMessage("m-1", "first"), makeMessage("m-2", "second")],
    };
    rerender(<SessionPage />);

    expect(screen.getByTestId("message-count")).toHaveTextContent("2");
    expect(screen.getByTestId("message-content")).toHaveTextContent("first|second");
  });

  it("clears the previous session state before hydrating the next transcript", () => {
    transcriptState = {
      ...transcriptState,
      transcriptMessages: [makeMessage("a-1", "from-a")],
    };

    const { rerender } = render(<SessionPage />);
    expect(screen.getByTestId("chat-header")).toHaveTextContent("alpha:true");
    expect(screen.getByTestId("message-content")).toHaveTextContent("from-a");

    routeParams = { id: "sess-2" };
    sessionState = {
      data: makeSession("sess-2"),
      isLoading: false,
      error: null,
    };
    transcriptState = {
      transcriptMessages: undefined,
      isLoadingTranscript: false,
      error: null,
    };
    rerender(<SessionPage />);

    expect(screen.getByTestId("message-count")).toHaveTextContent("0");
    expect(screen.getByTestId("message-content")).toHaveTextContent("");

    transcriptState = {
      ...transcriptState,
      transcriptMessages: [makeMessage("b-1", "from-b")],
    };
    rerender(<SessionPage />);

    expect(screen.getByTestId("message-count")).toHaveTextContent("1");
    expect(screen.getByTestId("message-content")).toHaveTextContent("from-b");
  });

  it("hides the composer until the session is active", () => {
    sessionState = {
      data: {
        ...makeSession("sess-1"),
        state: "starting",
      },
      isLoading: false,
      error: null,
    };

    render(<SessionPage />);

    expect(screen.queryByTestId("message-composer")).not.toBeInTheDocument();
  });

  it("renders the composer inert when a permission prompt is pending", () => {
    render(<SessionPage />);

    act(() => {
      useSessionStore.getState().setPendingPermission({
        requestId: "req-1",
        toolName: "Bash",
        toolInput: {},
        action: "execute",
        resource: "cmd",
      });
    });

    expect(screen.getByTestId("message-composer")).toHaveAttribute("data-inert", "true");
    expect(screen.getByTestId("permission-prompt")).toBeInTheDocument();
  });
});
