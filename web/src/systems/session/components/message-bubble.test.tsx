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

vi.mock("@/lib/utils", async importActual => {
  const actual = await importActual<typeof import("@/lib/utils")>();
  return {
    ...actual,
    cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
  };
});

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
  it("renders user message right-aligned with surface-elevated bubble", async () => {
    render(<MessageBubble message={makeMessage({ role: "user", content: "Hello" })} />);
    const bubble = screen.getByTestId("message-bubble-user");
    expect(bubble).toBeInTheDocument();
    expect(bubble.getAttribute("data-slot")).toBe("chat-message");
    expect(bubble.getAttribute("data-align")).toBe("right");
    expect(bubble.className).toContain("justify-end");

    const body = bubble.querySelector<HTMLElement>('[data-slot="chat-message-body"]');
    expect(body).not.toBeNull();
    expect(body?.className).toMatch(/bg-\[color:var\(--color-surface-elevated\)\]/);
    expect(body?.className).toContain("rounded-[var(--radius-lg)]");

    expect(await screen.findByText("Hello")).toBeInTheDocument();
    expect(screen.getByTestId("user-bubble")).toBeInTheDocument();
  });

  it("renders user meta slot with uppercase YOU + formatted timestamp", () => {
    const stamp = Date.parse("2026-04-18T12:30:00Z");
    render(
      <MessageBubble message={makeMessage({ role: "user", content: "hi", timestamp: stamp })} />
    );
    const meta = screen
      .getByTestId("message-bubble-user")
      .querySelector<HTMLElement>('[data-slot="chat-message-meta"]');
    expect(meta).not.toBeNull();
    expect(meta?.textContent).toMatch(/YOU/);
    expect(meta?.textContent).toMatch(/\d{1,2}:\d{2}/);
    expect(meta?.className).toMatch(/uppercase/);
  });

  it("renders agent (assistant) message left-aligned with no bubble background", async () => {
    render(<MessageBubble message={makeMessage({ content: "Hi there" })} />);
    const bubble = screen.getByTestId("message-bubble-assistant");
    expect(bubble).toBeInTheDocument();
    expect(bubble.getAttribute("data-slot")).toBe("chat-message");
    expect(bubble.getAttribute("data-align")).toBe("left");
    expect(bubble.className).not.toContain("justify-end");
    expect(await screen.findByText("Hi there")).toBeInTheDocument();
  });

  it("renders agent label with 8px success StatusDot and mono agent name", () => {
    render(<MessageBubble message={makeMessage({ content: "Hello" })} agentName="claude-code" />);

    const label = screen.getByTestId("agent-label");
    expect(label).toBeInTheDocument();

    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.getAttribute("data-slot")).toBe("status-dot");
    expect(dot.getAttribute("data-tone")).toBe("success");
    expect(dot.getAttribute("data-size")).toBe("md");

    const agentNameEl = label.querySelector(".font-mono");
    expect(agentNameEl).toBeInTheDocument();
    expect(agentNameEl?.textContent).toBe("claude-code");
  });

  it("uses accent tone + pulse for streaming agent message", () => {
    render(<MessageBubble message={makeMessage({ content: "", isStreaming: true })} />);
    const dot = screen.getByTestId("agent-status-dot");
    expect(dot.getAttribute("data-tone")).toBe("accent");
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

  it("renders a system message as a full-width divider row without bubble wrapper", () => {
    render(
      <MessageBubble
        message={makeMessage({ role: "system", content: "Session resumed from checkpoint 8471." })}
      />
    );
    const shell = screen.getByTestId("message-bubble-system");
    expect(shell).toBeInTheDocument();
    expect(shell.getAttribute("data-role")).toBe("system");
    // ChatMessageBubble role="system" wraps body with `h-px flex-1` dividers.
    const dividers = shell.querySelectorAll('span[aria-hidden="true"]');
    expect(dividers.length).toBeGreaterThanOrEqual(2);
    expect(shell.textContent).toContain("Session resumed from checkpoint 8471.");
  });

  it("renders a diff message with CodeBlock + optional language + path + ± summary", () => {
    render(
      <MessageBubble
        message={makeMessage({
          role: "diff",
          content: "",
          diff: {
            language: "ts",
            content: "- old line\n+ new line",
            path: "packages/runtime/src/session/stream.ts",
            additions: 4,
            removals: 38,
          },
        })}
      />
    );
    const shell = screen.getByTestId("message-bubble-diff");
    expect(shell).toBeInTheDocument();
    expect(shell.getAttribute("data-role")).toBe("diff");

    const code = screen.getByTestId("message-bubble-diff-code");
    expect(code.getAttribute("data-slot")).toBe("code-block");
    const eyebrow = code.querySelector<HTMLElement>('[data-slot="code-block-language"]');
    expect(eyebrow?.textContent).toBe("ts");
    expect(code.textContent).toContain("old line");
    expect(code.textContent).toContain("new line");

    expect(
      shell.querySelector<HTMLElement>('[data-slot="chat-message-diff-path"]')?.textContent
    ).toBe("packages/runtime/src/session/stream.ts");
    expect(
      shell.querySelector<HTMLElement>('[data-slot="chat-message-diff-summary"]')?.textContent
    ).toMatch(/\+4/);
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
    expect(screen.getByTestId("thinking-trigger")).toBeInTheDocument();
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
