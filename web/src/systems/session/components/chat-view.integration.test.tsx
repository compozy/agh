import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

import type { UIMessage } from "../types";

// Mock react-virtual to avoid needing real scroll measurements
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
    ...props
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    [key: string]: unknown;
  }) => (
    <button onClick={onClick} {...props}>
      {children}
    </button>
  ),
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

import { ChatView } from "./chat-view";

function makeMessage(
  overrides: Partial<UIMessage> & { id: string; role: UIMessage["role"] }
): UIMessage {
  return {
    content: "",
    timestamp: Date.now(),
    ...overrides,
  };
}

describe("ChatView integration", () => {
  beforeEach(() => {
    // Reset scroll-related mocks
    vi.restoreAllMocks();
  });

  it("renders user and assistant messages", async () => {
    const messages: UIMessage[] = [
      makeMessage({ id: "m1", role: "user", content: "What is 2+2?" }),
      makeMessage({ id: "m2", role: "assistant", content: "The answer is 4." }),
    ];

    render(<ChatView messages={messages} isStreaming={false} />);

    expect(await screen.findByText("What is 2+2?")).toBeInTheDocument();
    expect(await screen.findByText("The answer is 4.")).toBeInTheDocument();
  });

  it("renders empty state when no messages", () => {
    render(<ChatView messages={[]} isStreaming={false} />);

    expect(screen.getByTestId("chat-empty-state")).toBeInTheDocument();
    expect(screen.getByText("Send a message to start the conversation")).toBeInTheDocument();
  });

  it("renders processing indicator when streaming with no active content", () => {
    const messages: UIMessage[] = [
      makeMessage({ id: "m1", role: "user", content: "Do something" }),
    ];

    render(<ChatView messages={messages} isStreaming={true} />);

    expect(screen.getByTestId("processing-indicator")).toBeInTheDocument();
  });

  it("does not show processing indicator when assistant is streaming content", () => {
    const messages: UIMessage[] = [
      makeMessage({ id: "m1", role: "user", content: "Hello" }),
      makeMessage({
        id: "m2",
        role: "assistant",
        content: "I am responding...",
        isStreaming: true,
      }),
    ];

    render(<ChatView messages={messages} isStreaming={true} />);

    expect(screen.queryByTestId("processing-indicator")).not.toBeInTheDocument();
  });

  it("renders tool group with tool cards for consecutive tool messages", () => {
    const messages: UIMessage[] = [
      makeMessage({ id: "m1", role: "user", content: "Read a file" }),
      makeMessage({
        id: "m2",
        role: "tool_call",
        toolName: "Read",
        toolInput: { file_path: "/src/main.ts" },
      }),
      makeMessage({
        id: "m2",
        role: "tool_result",
        toolResult: { stdout: "const x = 1;\n" },
      }),
      makeMessage({ id: "m4", role: "assistant", content: "Here is the file." }),
    ];

    render(<ChatView messages={messages} isStreaming={false} />);

    expect(screen.getByTestId("tool-group")).toBeInTheDocument();
    expect(screen.getByTestId("tool-call-card")).toBeInTheDocument();
    // Should show past-tense label since result is present
    expect(screen.getByText("Read file")).toBeInTheDocument();
  });

  it("renders tool group with multiple tool cards", () => {
    const messages: UIMessage[] = [
      makeMessage({
        id: "tc-1",
        role: "tool_call",
        toolName: "Read",
        toolInput: { file_path: "/a.ts" },
      }),
      makeMessage({
        id: "tc-1",
        role: "tool_result",
        toolResult: { stdout: "content" },
      }),
      makeMessage({
        id: "tc-2",
        role: "tool_call",
        toolName: "Bash",
        toolInput: { command: "ls" },
      }),
      makeMessage({
        id: "tc-2",
        role: "tool_result",
        toolResult: { stdout: "output" },
      }),
    ];

    render(<ChatView messages={messages} isStreaming={false} />);

    const cards = screen.getAllByTestId("tool-call-card");
    expect(cards).toHaveLength(2);
    expect(screen.getByText("Read file")).toBeInTheDocument();
    expect(screen.getByText("Ran command")).toBeInTheDocument();
  });

  it("renders executing tool card when result not yet available", () => {
    const messages: UIMessage[] = [
      makeMessage({
        id: "tc-running",
        role: "tool_call",
        toolName: "Bash",
        toolInput: { command: "make build" },
      }),
    ];

    render(<ChatView messages={messages} isStreaming={true} />);

    expect(screen.getByTestId("tool-card-executing")).toBeInTheDocument();
    expect(screen.getByText("Running...")).toBeInTheDocument();
  });

  it("renders thinking block when message has thinking content", async () => {
    const messages: UIMessage[] = [
      makeMessage({
        id: "m1",
        role: "assistant",
        content: "The answer",
        thinking: "Let me reason about this...",
        thinkingComplete: true,
      }),
    ];

    render(<ChatView messages={messages} isStreaming={false} />);

    expect(await screen.findByText("Thought process")).toBeInTheDocument();
  });

  it("renders multiple messages in correct order", () => {
    const messages: UIMessage[] = [
      makeMessage({ id: "m1", role: "user", content: "First question" }),
      makeMessage({ id: "m2", role: "assistant", content: "First answer" }),
      makeMessage({ id: "m3", role: "user", content: "Second question" }),
      makeMessage({ id: "m4", role: "assistant", content: "Second answer" }),
    ];

    render(<ChatView messages={messages} isStreaming={false} />);

    const userBubbles = screen.getAllByTestId("message-bubble-user");
    const assistantBubbles = screen.getAllByTestId("message-bubble-assistant");
    expect(userBubbles).toHaveLength(2);
    expect(assistantBubbles).toHaveLength(2);
  });
});
