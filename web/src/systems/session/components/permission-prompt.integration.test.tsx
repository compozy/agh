import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

import type { UIMessage, PermissionRequest } from "../types";
import { useSessionStore } from "../hooks/use-session-store";

// Mock dependencies for ChatView
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: ({
    count,
    getItemKey,
  }: {
    count: number;
    getItemKey: (i: number) => string;
  }) => ({
    getVirtualItems: () =>
      Array.from({ length: count }, (_, i) => ({
        index: i,
        key: getItemKey(i),
        start: i * 60,
        size: 60,
      })),
    getTotalSize: () => count * 60,
    measureElement: () => {},
  }),
}));

vi.mock("@/lib/utils", async importActual => {
  const actual = await importActual<typeof import("@/lib/utils")>();
  return {
    ...actual,
    cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
  };
});

vi.mock("react-syntax-highlighter", () => ({
  PrismAsyncLight: Object.assign(({ children }: { children: string }) => <pre>{children}</pre>, {
    registerLanguage: vi.fn(),
  }),
}));

vi.mock("react-syntax-highlighter/dist/esm/styles/prism", () => ({
  oneDark: {},
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn() },
}));

vi.mock("../adapters/session-api", () => ({
  approveSession: vi.fn().mockResolvedValue(undefined),
}));

import { ChatView } from "./chat-view";
import { PermissionPrompt } from "./permission-prompt";
import { MessageComposer } from "./message-composer";

function makeMessage(
  overrides: Partial<UIMessage> & { id: string; role: UIMessage["role"] }
): UIMessage {
  return {
    content: "",
    timestamp: Date.now(),
    ...overrides,
  };
}

const mockPermission: PermissionRequest = {
  requestId: "req-123",
  toolName: "Bash",
  action: "execute",
  resource: "rm -rf /tmp/test",
  toolInput: { command: "rm -rf /tmp/test" },
};

describe("Permission prompt integration", () => {
  beforeEach(() => {
    useSessionStore.setState({
      activeSessionId: null,
      historyMessages: [],
      liveMessages: [],
      isStreaming: false,
      awaitingTranscriptSync: false,
      pendingPermission: null,
    });
  });

  it("permission prompt appears when pendingPermission is set in store", async () => {
    const messages: UIMessage[] = [
      makeMessage({ id: "m1", role: "user", content: "Do something dangerous" }),
    ];

    // Simulate the session page layout
    const { rerender } = render(
      <div>
        <ChatView messages={messages} isStreaming={false} />
        <MessageComposer onSend={vi.fn()} disabled={false} />
      </div>
    );

    // No permission prompt initially
    expect(screen.queryByTestId("permission-prompt")).not.toBeInTheDocument();

    // Set pending permission in store
    useSessionStore.getState().setPendingPermission(mockPermission);

    // Re-render with permission prompt (as session page would)
    rerender(
      <div>
        <ChatView messages={messages} isStreaming={false} />
        <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={vi.fn()} />
        <MessageComposer onSend={vi.fn()} inert />
      </div>
    );

    expect(screen.getByTestId("permission-prompt")).toBeInTheDocument();
    expect(await screen.findByText("Permission Required")).toBeInTheDocument();
    expect(await screen.findByText("Bash")).toBeInTheDocument();
  });

  it("composer is disabled while permission is pending", () => {
    const messages: UIMessage[] = [makeMessage({ id: "m1", role: "user", content: "Hello" })];

    useSessionStore.getState().setPendingPermission(mockPermission);

    render(
      <div>
        <ChatView messages={messages} isStreaming={false} />
        <PermissionPrompt permission={mockPermission} sessionId="sess-001" onResolved={vi.fn()} />
        <MessageComposer onSend={vi.fn()} inert />
      </div>
    );

    const textarea = screen.getByTestId("composer-textarea");
    expect(textarea).toBeDisabled();

    const sendButton = screen.getByTestId("composer-send-button");
    expect(sendButton).toBeDisabled();
  });
});
