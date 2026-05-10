import { render, screen, act } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

import type { UIMessage } from "../../types";

vi.mock("@/lib/utils", async importActual => {
  const actual = await importActual<typeof import("@/lib/utils")>();
  return {
    ...actual,
    cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
  };
});

vi.mock("@agh/ui", async () => {
  const actual = await vi.importActual<typeof import("@agh/ui")>("@agh/ui");
  return {
    ...actual,
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
  };
});

import { ToolCallCard } from "../tool-call-card";

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

function queryPrimitiveRoot(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
}

function queryStatusBadge(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card-status"]');
}

describe("ToolCallCard", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the primitive card shell with surface bg + token radius", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const root = queryPrimitiveRoot();
    expect(root).not.toBeNull();
    expect(root?.className).toContain("bg-(--canvas-soft)");
    expect(root?.className).toContain("border-(--line)");
    expect(root?.className).toContain("rounded-md");
  });

  it("renders a tool icon in the primitive header", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(screen.getByTestId("tool-call-icon")).toBeInTheDocument();
  });

  it("shows tool name in executing state", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const executing = screen.getByTestId("tool-card-executing");
    expect(executing).toBeInTheDocument();
    expect(executing).toHaveTextContent("Reading...");
  });

  it("renders RUNNING status badge with accent tone for executing tool", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const badge = queryStatusBadge();
    expect(badge).not.toBeNull();
    expect(badge?.textContent).toBe("RUNNING");
    expect(badge?.getAttribute("data-tone")).toBe("accent");
    expect(badge?.className).toMatch(/bg-\(--accent-tint\)/);
    expect(queryPrimitiveRoot()?.getAttribute("data-status")).toBe("running");
  });

  it("renders DONE status badge with success tone for completed tool", () => {
    render(<ToolCallCard message={makeToolMessage({ toolResult: { content: "file content" } })} />);
    const badge = queryStatusBadge();
    expect(badge?.textContent).toBe("DONE");
    expect(badge?.getAttribute("data-tone")).toBe("success");
    expect(badge?.className).toMatch(/bg-\(--success-tint\)/);
    expect(queryPrimitiveRoot()?.getAttribute("data-status")).toBe("done");
  });

  it("renders ERROR status badge with danger tone and danger-toned card border for failed tool", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({
          toolResult: { error: "not found" },
          toolError: true,
        })}
      />
    );
    const badge = queryStatusBadge();
    expect(badge?.textContent).toBe("ERROR");
    expect(badge?.getAttribute("data-tone")).toBe("danger");
    expect(badge?.className).toMatch(/bg-\(--danger-tint\)/);
    const root = queryPrimitiveRoot();
    expect(root?.getAttribute("data-status")).toBe("error");
    expect(root?.className).toContain("data-[status=error]:border-(--danger)/40");
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
