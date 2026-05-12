import { fireEvent, render, screen } from "@testing-library/react";
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

function queryRoot(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
}

function queryStatusPill(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card-status"]');
}

function queryToolName(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card-tool"]');
}

function queryInputRegion(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card-input"]');
}

function queryOutputRegion(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card-output"]');
}

describe("Session ToolCallCard — wraps <ToolCallCard> from @agh/ui", () => {
  it("Should expose the tool name through the header slot", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(queryToolName()).toHaveTextContent("Read");
  });

  it("Should map in-flight (no result, no error) to status=in_progress (info tone)", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("in_progress");
    expect(queryStatusPill()?.getAttribute("data-tone")).toBe("info");
    expect(screen.getByTestId("tool-card-executing")).toHaveTextContent("Reading...");
  });

  it("Should map result present + no error to status=completed (success tone)", () => {
    render(<ToolCallCard message={makeToolMessage({ toolResult: { content: "file" } })} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("completed");
    expect(queryStatusPill()?.getAttribute("data-tone")).toBe("success");
    expect(screen.getByTestId("tool-card-success")).toHaveTextContent("Read file");
  });

  it("Should map toolError to status=failed (danger tone) with the failure ring + errorMessage slot", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({ toolResult: { error: "not found" }, toolError: true })}
      />
    );
    expect(queryRoot()?.getAttribute("data-status")).toBe("failed");
    expect(queryStatusPill()?.getAttribute("data-tone")).toBe("danger");
    const errorNode = document.querySelector('[data-slot="tool-call-card-error"]');
    expect(errorNode?.textContent).toContain("Failed to read file");
    expect(screen.getByTestId("tool-card-error")).toHaveTextContent("Failed to read file");
  });

  it("Should render the Input section closed by default and toggle open on click", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const region = queryInputRegion();
    expect(region).not.toBeNull();
    expect(region?.getAttribute("data-open")).toBe("false");
    const toggle = region?.querySelector<HTMLButtonElement>(
      '[data-slot="tool-call-card-input-toggle"]'
    );
    expect(toggle?.getAttribute("aria-expanded")).toBe("false");
    fireEvent.click(toggle!);
    expect(region?.getAttribute("data-open")).toBe("true");
  });

  it("Should render the Output section closed by default once a result is available", () => {
    render(<ToolCallCard message={makeToolMessage({ toolResult: { content: "abc" } })} />);
    const region = queryOutputRegion();
    expect(region).not.toBeNull();
    expect(region?.getAttribute("data-open")).toBe("false");
    expect(
      region
        ?.querySelector('[data-slot="tool-call-card-output-toggle"]')
        ?.getAttribute("aria-expanded")
    ).toBe("false");
  });

  it("Should not render an Output region while the tool is still running", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(queryOutputRegion()).toBeNull();
  });
});
