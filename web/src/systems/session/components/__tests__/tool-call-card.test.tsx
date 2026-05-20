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

function queryStatusIndicator(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card-status"]');
}

function queryToolName(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card-tool"]');
}

function queryInputToggle(): HTMLButtonElement | null {
  return document.querySelector<HTMLButtonElement>('[data-slot="tool-call-card-input-toggle"]');
}

function queryOutputToggle(): HTMLButtonElement | null {
  return document.querySelector<HTMLButtonElement>('[data-slot="tool-call-card-output-toggle"]');
}

function queryOutputRegion(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-card-output"]');
}

describe("Session ToolCallCard — wraps <ToolCallCard> from @agh/ui", () => {
  it("Should expose the tool name through the header slot", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(queryToolName()).toHaveTextContent("Read");
  });

  it("Should map in-flight (no result, no error) to status=in_progress with a Spinner", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("in_progress");
    const indicator = queryStatusIndicator();
    expect(indicator?.getAttribute("data-status")).toBe("in_progress");
    expect(indicator?.getAttribute("aria-label")).toBe("Running");
    expect(indicator?.classList.contains("animate-spin")).toBe(true);
    expect(screen.getByTestId("tool-card-executing")).toHaveTextContent("Reading...");
  });

  it("Should map result present + no error to status=completed (success icon)", () => {
    render(<ToolCallCard message={makeToolMessage({ toolResult: { content: "file" } })} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("completed");
    const indicator = queryStatusIndicator();
    expect(indicator?.getAttribute("data-status")).toBe("completed");
    expect(indicator?.getAttribute("aria-label")).toBe("Done");
    expect(indicator?.classList.contains("text-success")).toBe(true);
    expect(screen.getByTestId("tool-card-success")).toHaveTextContent("Read file");
  });

  it("Should map toolError to status=failed (danger icon) with the failure ring + errorMessage slot", () => {
    render(
      <ToolCallCard
        message={makeToolMessage({ toolResult: { error: "not found" }, toolError: true })}
      />
    );
    expect(queryRoot()?.getAttribute("data-status")).toBe("failed");
    const indicator = queryStatusIndicator();
    expect(indicator?.getAttribute("data-status")).toBe("failed");
    expect(indicator?.getAttribute("aria-label")).toBe("Error");
    expect(indicator?.classList.contains("text-danger")).toBe(true);
    const errorNode = document.querySelector('[data-slot="tool-call-card-error"]');
    expect(errorNode?.textContent).toContain("Failed to read file");
    expect(screen.getByTestId("tool-card-error")).toHaveTextContent("Failed to read file");
  });

  it("Should render the Input chip closed by default and toggle open on click", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    const toggle = queryInputToggle();
    expect(toggle).not.toBeNull();
    expect(toggle?.getAttribute("aria-expanded")).toBe("false");
    expect(toggle?.getAttribute("data-open")).toBe("false");
    fireEvent.click(toggle!);
    expect(
      document.querySelector('[data-slot="tool-call-card-input"]')?.getAttribute("data-open")
    ).toBe("true");
  });

  it("Should render the Output chip closed by default once a result is available", () => {
    render(<ToolCallCard message={makeToolMessage({ toolResult: { content: "abc" } })} />);
    const toggle = queryOutputToggle();
    expect(toggle).not.toBeNull();
    expect(toggle?.getAttribute("aria-expanded")).toBe("false");
  });

  it("Should not render an Output chip while the tool is still running", () => {
    render(<ToolCallCard message={makeToolMessage()} />);
    expect(queryOutputToggle()).toBeNull();
    expect(queryOutputRegion()).toBeNull();
  });
});
