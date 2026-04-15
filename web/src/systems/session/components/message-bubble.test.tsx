import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { UIMessage } from "../types";

vi.mock("react-syntax-highlighter", () => ({
  PrismAsyncLight: Object.assign(
    ({ children }: { children: string }) => <pre data-testid="syntax-highlighter">{children}</pre>,
    {
      registerLanguage: vi.fn(),
    }
  ),
}));

vi.mock("react-syntax-highlighter/dist/esm/styles/prism", () => ({
  oneDark: {},
}));

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@/components/ui/collapsible", () => ({
  Collapsible: ({ children, ...props }: Record<string, unknown>) => (
    <div data-testid="collapsible" {...props}>
      {children as React.ReactNode}
    </div>
  ),
  CollapsibleTrigger: ({ children, ...props }: Record<string, unknown>) => (
    <button data-testid="collapsible-trigger" {...props}>
      {children as React.ReactNode}
    </button>
  ),
  CollapsibleContent: ({ children }: Record<string, unknown>) => (
    <div data-testid="collapsible-content">{children as React.ReactNode}</div>
  ),
}));

import { MessageBubble } from "./message-bubble";

function makeMessage(overrides: Partial<UIMessage> = {}): UIMessage {
  return {
    id: "msg-1",
    role: "assistant",
    content: "",
    timestamp: Date.now(),
    ...overrides,
  };
}

describe("MessageBubble", () => {
  it("renders user message right-aligned with bubble background", async () => {
    render(<MessageBubble message={makeMessage({ role: "user", content: "Hello" })} />);
    const wrapper = screen.getByTestId("message-bubble-user");
    expect(wrapper).toBeInTheDocument();
    expect(wrapper.className).toContain("justify-end");

    const bubble = screen.getByTestId("user-bubble");
    expect(bubble.className).toMatch(/bg-\[color:var\(--color-surface-elevated\)\]/);
    expect(bubble.className).toContain("rounded-2xl");
    expect(await screen.findByText("Hello")).toBeInTheDocument();
  });

  it("renders agent message left-aligned with no bubble background", async () => {
    render(<MessageBubble message={makeMessage({ content: "Hi there" })} />);
    const wrapper = screen.getByTestId("message-bubble-assistant");
    expect(wrapper).toBeInTheDocument();
    expect(wrapper.className).not.toContain("justify-end");
    expect(await screen.findByText("Hi there")).toBeInTheDocument();
  });

  it("renders agent label with JetBrains Mono uppercase name and status dot", () => {
    render(<MessageBubble message={makeMessage({ content: "Hello" })} agentName="claude-code" />);

    const label = screen.getByTestId("agent-label");
    expect(label).toBeInTheDocument();

    const agentNameEl = label.querySelector(".font-mono");
    expect(agentNameEl).toBeInTheDocument();
    expect(agentNameEl?.className).toContain("uppercase");
    expect(agentNameEl?.textContent).toBe("claude-code");

    // Status dot
    const dot = label.querySelector(".rounded-full");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toMatch(/bg-\[color:var\(--color-success\)\]/);
  });

  it("shows default agent name when agentName prop is not provided", () => {
    render(<MessageBubble message={makeMessage({ content: "Hello" })} />);
    const label = screen.getByTestId("agent-label");
    expect(label.textContent).toContain("Agent");
  });

  it("omits invalid timestamps from the agent label", () => {
    render(<MessageBubble message={makeMessage({ content: "Hello", timestamp: 0 })} />);
    const label = screen.getByTestId("agent-label");
    expect(label.textContent).not.toContain("Invalid Date");
    expect(label.textContent).not.toContain("1970");
  });

  it("renders markdown headings", async () => {
    render(<MessageBubble message={makeMessage({ content: "# Heading 1\n\nSome text" })} />);
    expect(await screen.findByRole("heading", { level: 1 })).toHaveTextContent("Heading 1");
  });

  it("renders markdown code blocks with syntax highlighter", async () => {
    const content = "```javascript\nconst x = 1;\n```";
    render(<MessageBubble message={makeMessage({ content })} />);
    expect(await screen.findByTestId("syntax-highlighter")).toBeInTheDocument();
    expect(await screen.findByText("const x = 1;")).toBeInTheDocument();
  });

  it("renders markdown links", async () => {
    render(
      <MessageBubble message={makeMessage({ content: "[click here](https://example.com)" })} />
    );
    const link = await screen.findByRole("link", { name: "click here" });
    expect(link).toHaveAttribute("href", "https://example.com");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders inline code", async () => {
    render(<MessageBubble message={makeMessage({ content: "Use `foo()` to do that" })} />);
    expect(await screen.findByText("foo()")).toBeInTheDocument();
  });

  it("renders thinking block when thinking is present", () => {
    render(
      <MessageBubble
        message={makeMessage({ thinking: "Let me think about this...", thinkingComplete: true })}
      />
    );
    expect(screen.getByTestId("collapsible")).toBeInTheDocument();
  });

  it("shows streaming placeholder when no content and isStreaming", () => {
    render(<MessageBubble message={makeMessage({ content: "", isStreaming: true })} />);
    expect(screen.getByText("...")).toBeInTheDocument();
  });

  it("does not render a copy button for empty user messages", () => {
    render(<MessageBubble message={makeMessage({ role: "user", content: "" })} />);
    expect(screen.queryByRole("button", { name: "Copy message" })).not.toBeInTheDocument();
  });

  it("does not re-render when content is unchanged (memo check)", async () => {
    const message = makeMessage({ content: "Hello" });
    const { rerender } = render(<MessageBubble message={message} />);
    rerender(<MessageBubble message={message} />);
    expect(await screen.findByText("Hello")).toBeInTheDocument();
  });
});
