import { render, screen, act } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

import type { UIMessage } from "../types";

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="tooltip">{children}</div>
  ),
  TooltipTrigger: ({ children, ...props }: Record<string, unknown>) => (
    <div data-testid="tooltip-trigger" {...props}>
      {children as React.ReactNode}
    </div>
  ),
  TooltipContent: ({ children, ...props }: Record<string, unknown>) => (
    <div data-testid="tooltip-content" {...props}>
      {children as React.ReactNode}
    </div>
  ),
}));

import { ToolCallCard } from "./tool-call-card";

function makeToolMessage(overrides: Partial<UIMessage> = {}): UIMessage {
  return {
    id: "tc-1",
    role: "tool_call",
    content: "",
    toolName: "Read",
    toolInput: { file_path: "/src/main.ts" },
    timestamp: Date.now(),
    ...overrides,
  };
}

describe("ToolCallCard", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders with card styling (border and surface background)", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const trigger = screen.getByTestId("tool-card-trigger");
    expect(trigger.className).toMatch(/border-\[color:var\(--color-divider\)\]/);
    expect(trigger.className).toMatch(/bg-\[color:var\(--color-surface\)\]/);
    expect(trigger.className).toContain("rounded-lg");
  });

  it("renders terminal icon for tool", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(screen.getByTestId("tool-call-icon")).toBeInTheDocument();
  });

  it("shows tool name in executing state", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const executing = screen.getByTestId("tool-card-executing");
    expect(executing).toBeInTheDocument();
    expect(executing).toHaveTextContent("Reading...");
  });

  it("renders RUNNING status badge with accent color for executing tool", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const badge = screen.getByTestId("tool-status-badge-running");
    expect(badge).toHaveTextContent("Running");
    expect(badge.className).toMatch(/bg-\[color:var\(--color-accent-tint\)\]/);
    expect(badge.className).toMatch(/text-\[color:var\(--color-accent\)\]/);
  });

  it("renders DONE status badge with green color for completed tool", () => {
    render(<ToolCallCard message={makeToolMessage({ toolResult: { content: "file content" } })} />);
    const badge = screen.getByTestId("tool-status-badge-done");
    expect(badge).toHaveTextContent("Done");
    expect(badge.className).toMatch(/bg-\[color:var\(--color-success-tint\)\]/);
    expect(badge.className).toMatch(/text-\[color:var\(--color-success\)\]/);
  });

  it("renders ERROR status badge with red color for failed tool", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({
          toolResult: { error: "not found" },
          toolError: true,
        })}
      />
    );
    const badge = screen.getByTestId("tool-status-badge-error");
    expect(badge).toHaveTextContent("Error");
    expect(badge.className).toMatch(/bg-\[color:var\(--color-danger-tint\)\]/);
    expect(badge.className).toMatch(/text-\[color:var\(--color-danger\)\]/);
  });

  it("renders success state with past-tense label for completed tool", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({
          toolResult: { content: "file content" },
        })}
      />
    );
    expect(screen.getByTestId("tool-card-success")).toHaveTextContent("Read file");
  });

  it("renders error state label for failed tool", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({
          toolResult: { error: "not found" },
          toolError: true,
        })}
      />
    );
    expect(screen.getByTestId("tool-card-error")).toHaveTextContent("Failed to read file");
  });

  it("shows compact summary from tool input", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(screen.getByText("/src/main.ts")).toBeInTheDocument();
    expect(screen.queryByTestId("tooltip-content")).not.toBeInTheDocument();
  });

  it("auto-expands when toolResult arrives", () => {
    const msg = makeToolMessage();
    const { rerender } = render(<ToolCallCard message={msg} />);
    expect(screen.queryByTestId("tool-card-expanded")).not.toBeInTheDocument();

    rerender(<ToolCallCard message={{ ...msg, toolResult: { content: "file content" } }} />);
    expect(screen.getByTestId("tool-card-expanded")).toBeInTheDocument();
  });

  it("auto-collapses after 2s when not manually toggled", () => {
    const msg = makeToolMessage();
    const { rerender } = render(<ToolCallCard message={msg} />);

    rerender(<ToolCallCard message={{ ...msg, toolResult: { content: "file content" } }} />);
    expect(screen.getByTestId("tool-card-expanded")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(screen.queryByTestId("tool-card-expanded")).not.toBeInTheDocument();
  });

  it("does not auto-expand for Edit/Write tools (they start expanded)", () => {
    const msg = makeToolMessage({
      id: "tc-edit-1",
      toolName: "Edit",
      toolInput: { file_path: "/a.ts", old_string: "a", new_string: "b" },
    });
    const { rerender } = render(<ToolCallCard message={msg} />);

    rerender(<ToolCallCard message={{ ...msg, toolResult: { content: "ok" } }} />);

    expect(screen.getByTestId("tool-card-expanded")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(2500);
    });
    expect(screen.getByTestId("tool-card-expanded")).toBeInTheDocument();
  });

  it("cancels auto-collapse when user manually toggles", () => {
    const msg = makeToolMessage();
    const { rerender } = render(<ToolCallCard message={msg} />);

    rerender(<ToolCallCard message={{ ...msg, toolResult: { content: "result" } }} />);
    expect(screen.getByTestId("tool-card-expanded")).toBeInTheDocument();

    act(() => {
      screen.getByTestId("tool-card-trigger").click();
    });
    expect(screen.queryByTestId("tool-card-expanded")).not.toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(2500);
    });
    expect(screen.queryByTestId("tool-card-expanded")).not.toBeInTheDocument();
  });

  it("renders Bash tool with correct labels", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({
          toolName: "Bash",
          toolInput: { command: "ls -la" },
        })}
      />
    );
    expect(screen.getByTestId("tool-card-executing")).toHaveTextContent("Running...");
    expect(screen.getByText("ls -la")).toBeInTheDocument();
  });

  it("shows the full tooltip content for truncated non-Bash summaries", () => {
    const longPath =
      "/very/long/project/path/with/many/segments/that/needs/a/tooltip/example-file.tsx";

    render(
      <ToolCallCard
        message={makeToolMessage({
          toolName: "Read",
          toolInput: { file_path: longPath },
        })}
      />
    );

    expect(screen.getByTestId("tooltip-trigger")).toHaveTextContent("\u2026");
    expect(screen.getByTestId("tooltip-content")).toHaveTextContent(longPath);
  });

  it("renders unknown tool with fallback labels", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({
          toolName: "CustomTool",
          toolInput: {},
          toolResult: { content: "done" },
        })}
      />
    );
    expect(screen.getByTestId("tool-card-success")).toHaveTextContent("Used CustomTool");
  });
});
