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

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@agh/ui", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    ...props
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    [key: string]: unknown;
  }) => (
    <button onClick={onClick} disabled={disabled} {...props}>
      {children}
    </button>
  ),
  Card: ({ children, ...props }: Record<string, unknown>) => (
    <div {...props}>{children as React.ReactNode}</div>
  ),
  CardHeader: ({ children }: Record<string, unknown>) => <div>{children as React.ReactNode}</div>,
  CardTitle: ({ children }: Record<string, unknown>) => <h3>{children as React.ReactNode}</h3>,
  CardContent: ({ children }: Record<string, unknown>) => <div>{children as React.ReactNode}</div>,
  CardFooter: ({ children }: Record<string, unknown>) => <div>{children as React.ReactNode}</div>,
  Collapsible: ({ children }: Record<string, unknown>) => <div>{children as React.ReactNode}</div>,
  CollapsibleTrigger: ({ children }: Record<string, unknown>) => (
    <button>{children as React.ReactNode}</button>
  ),
  CollapsibleContent: ({ children }: Record<string, unknown>) => (
    <div>{children as React.ReactNode}</div>
  ),
}));

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
      messages: [],
      isStreaming: false,
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
        <MessageComposer onSend={vi.fn()} disabled={true} />
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
        <MessageComposer onSend={vi.fn()} disabled={true} />
      </div>
    );

    const textarea = screen.getByTestId("composer-textarea");
    expect(textarea).toBeDisabled();

    const sendButton = screen.getByTestId("composer-send-button");
    expect(sendButton).toBeDisabled();
  });
});
