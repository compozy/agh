import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { UIMessage } from "../../types";
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

function queryChatToolCardRoot(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="chat-tool-card"]');
}

function queryStatusPill(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="chat-tool-card-status"]');
}

function queryNameMonoId(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="chat-tool-card-name"]');
}

function queryInputRegion(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="chat-tool-card-input"]');
}

function queryOutputRegion(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="chat-tool-card-output"]');
}

describe("Session ToolCallCard — wraps <ChatToolCard> per ADR-014 §6", () => {
  it("Should render the primitive root with canvas-soft surface and token radius", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const root = queryChatToolCardRoot();
    expect(root).not.toBeNull();
    expect(root?.className).toContain("bg-(--canvas-soft)");
    expect(root?.className).toContain("rounded-(--radius-lg)");
  });

  it("Should expose the tool name through the head MonoId slot", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(queryNameMonoId()).toHaveTextContent("read");
  });

  it("Should map in-flight (no result, no error) to status=in_progress (info tone)", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const root = queryChatToolCardRoot();
    expect(root?.getAttribute("data-status")).toBe("in_progress");
    expect(queryStatusPill()?.getAttribute("data-tone")).toBe("info");
    expect(screen.getByTestId("tool-card-executing")).toHaveTextContent("Reading...");
  });

  it("Should map result present + no error to status=completed (success tone)", () => {
    render(<ToolCallCard message={makeToolMessage({ toolResult: { content: "file" } })} />);
    const root = queryChatToolCardRoot();
    expect(root?.getAttribute("data-status")).toBe("completed");
    expect(queryStatusPill()?.getAttribute("data-tone")).toBe("success");
    expect(screen.getByTestId("tool-card-success")).toHaveTextContent("Read file");
  });

  it("Should map toolError to status=failed (danger tone) with errorMessage + danger-tint background", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({ toolResult: { error: "not found" }, toolError: true })}
      />
    );
    const root = queryChatToolCardRoot();
    expect(root?.getAttribute("data-status")).toBe("failed");
    expect(root?.className).toContain("bg-(--danger-tint)");
    expect(queryStatusPill()?.getAttribute("data-tone")).toBe("danger");
    const errorNode = document.querySelector('[data-slot="chat-tool-card-error"]');
    expect(errorNode).not.toBeNull();
    expect(errorNode?.textContent).toContain("Failed to read file");
    expect(screen.getByTestId("tool-card-error")).toHaveTextContent("Failed to read file");
  });

  it("Should render a collapsible Input section when toolInput has entries", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const input = queryInputRegion();
    expect(input).not.toBeNull();
    const toggle = input?.querySelector('[data-slot="chat-tool-card-input-toggle"]');
    expect(toggle).not.toBeNull();
    expect(toggle?.getAttribute("aria-expanded")).toBe("true");
  });

  it("Should render a collapsible Output section once a result is available", () => {
    render(<ToolCallCard message={makeToolMessage({ toolResult: { content: "abc" } })} />);
    const output = queryOutputRegion();
    expect(output).not.toBeNull();
    const toggle = output?.querySelector('[data-slot="chat-tool-card-output-toggle"]');
    expect(toggle).not.toBeNull();
    expect(toggle?.getAttribute("aria-expanded")).toBe("true");
  });

  it("Should not render an Output region while the tool is still running", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(queryOutputRegion()).toBeNull();
  });
});
