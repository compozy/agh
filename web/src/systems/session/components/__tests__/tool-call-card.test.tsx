import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { UIMessage } from "../../types";
import { ToolCallRow } from "../tool-call-card";

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
  return document.querySelector<HTMLElement>('[data-slot="tool-call-row"]');
}

function queryStatusIndicator(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-row-status"]');
}

function queryToolName(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-row-tool"]');
}

function queryPreview(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-row-preview"]');
}

function queryIcon(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-row-icon"]');
}

function queryBody(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="tool-call-row-body"]');
}

describe("Session ToolCallRow — wraps <ToolCallRow> from @agh/ui", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("Should surface the tense-aware verb (not the raw tool name) in the row heading slot", () => {
    render(<ToolCallRow message={makeToolMessage()} />);
    // Read fixture is in-flight (no result) → active verb.
    expect(queryToolName()).toHaveTextContent("Reading...");
    expect(queryToolName()).not.toHaveTextContent("Read file");
  });

  it("Should render the mapped per-tool icon and never the terminal fallback for a known tool", () => {
    render(<ToolCallRow message={makeToolMessage({ toolResult: { content: "file" } })} />);
    const iconClass = queryIcon()?.getAttribute("class") ?? "";
    expect(iconClass).toContain("lucide-file-text");
    expect(iconClass).not.toContain("lucide-terminal");
  });

  it("Should show the compact input summary in the row preview slot", () => {
    render(<ToolCallRow message={makeToolMessage()} />);
    expect(queryPreview()).toHaveTextContent("/src/main.ts");
  });

  it("Should map Bash command summaries to the row preview slot", () => {
    const longCommand = "agh tool invoke agh__tool_info --input " + '{"tool_id":"agh__skill_view"}';
    render(
      <ToolCallRow
        message={makeToolMessage({
          toolName: "Bash",
          toolInput: { command: longCommand },
        })}
      />
    );
    expect(queryToolName()).toHaveTextContent("Running...");
    const preview = queryPreview();
    expect(preview).not.toBeNull();
    expect(preview?.textContent).toContain("agh tool invoke");
    expect(preview?.className).toContain("truncate");
  });

  it("Should render the pending row state (muted, no glyph) for an in-flight tool with empty input", () => {
    render(<ToolCallRow message={makeToolMessage({ toolInput: {} })} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("pending");
    expect(queryStatusIndicator()).toBeNull();
  });

  it("Should map an in-flight tool with input to the running row state with a Spinner", () => {
    render(<ToolCallRow message={makeToolMessage()} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("running");
    const indicator = queryStatusIndicator();
    expect(indicator?.getAttribute("data-status")).toBe("running");
    expect(indicator?.getAttribute("aria-label")).toBe("Running");
    expect(indicator?.getAttribute("class")).toContain("text-info");
    expect(screen.getByRole("status", { name: "Running" })).toBe(indicator);
    expect(queryToolName()).toHaveTextContent("Reading...");
  });

  it("Should map meaningful output to the success row state (Check, success tone)", () => {
    render(<ToolCallRow message={makeToolMessage({ toolResult: { content: "file" } })} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("success");
    const indicator = queryStatusIndicator();
    expect(indicator?.getAttribute("data-status")).toBe("success");
    expect(indicator?.getAttribute("aria-label")).toBe("Done");
    expect(indicator?.getAttribute("class")).toContain("text-success");
    expect(screen.getByRole("img", { name: "Done" })).toBe(indicator);
    expect(queryToolName()).toHaveTextContent("Read file");
  });

  it("Should render the empty row state (Minus, faint tone) for empty output mid-stream", () => {
    render(<ToolCallRow message={makeToolMessage({ toolResult: {} })} turnSettled={false} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("empty");
    const indicator = queryStatusIndicator();
    expect(indicator?.getAttribute("data-status")).toBe("empty");
    expect(indicator?.getAttribute("aria-label")).toBe("Empty");
    expect(indicator?.getAttribute("class")).toContain("text-faint");
  });

  it("Should promote a neutral tool to success only once the turn settles, never before", () => {
    const message = makeToolMessage({ toolResult: {} });
    const { rerender } = render(<ToolCallRow message={message} turnSettled={false} />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("empty");

    rerender(<ToolCallRow message={message} turnSettled />);
    expect(queryRoot()?.getAttribute("data-status")).toBe("success");
    expect(queryStatusIndicator()?.getAttribute("aria-label")).toBe("Done");
  });

  it("Should map a runtime error to failed, a danger heading, and the real error detail (not the verb)", () => {
    render(
      <ToolCallRow
        message={makeToolMessage({ toolResult: { error: "not found" }, toolError: true })}
      />
    );
    expect(queryRoot()?.getAttribute("data-status")).toBe("failed");
    const indicator = queryStatusIndicator();
    expect(indicator?.getAttribute("data-status")).toBe("failed");
    expect(indicator?.getAttribute("aria-label")).toBe("Error");
    expect(indicator?.getAttribute("class")).toContain("text-danger");
    expect(screen.getByRole("img", { name: "Error" })).toBe(indicator);
    expect(queryToolName()).toHaveTextContent("Failed to read file");
    expect(queryToolName()?.className).toContain("text-danger");
    expect(document.querySelector('[data-slot="tool-call-row-error"]')).toHaveTextContent(
      "not found"
    );
  });

  it("Should flag error-shaped output as failed while keeping the heading neutral (not a runtime error)", () => {
    render(
      <ToolCallRow
        message={makeToolMessage({
          toolName: "Bash",
          toolInput: { command: "deploy" },
          toolResult: { stderr: "bash: deploy: command not found" },
        })}
      />
    );
    expect(queryRoot()?.getAttribute("data-status")).toBe("failed");
    expect(queryStatusIndicator()?.getAttribute("aria-label")).toBe("Error");
    const headingEl = queryToolName();
    expect(headingEl).toHaveTextContent("Ran command");
    expect(headingEl?.className).not.toContain("text-danger");
    expect(headingEl?.className).toContain("text-muted");
  });

  it("Should toggle the inline Input body by click and keyboard", () => {
    render(<ToolCallRow message={makeToolMessage()} />);
    const trigger = screen.getByRole("button");
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    expect(queryBody()).toBeNull();

    fireEvent.click(trigger);
    expect(queryBody()).not.toBeNull();
    expect(document.querySelector('[data-slot="tool-call-row-input"]')).toHaveTextContent(
      '"file_path"'
    );

    fireEvent.keyDown(trigger, { key: "Enter" });
    expect(queryBody()).toBeNull();
    fireEvent.keyDown(trigger, { key: " " });
    expect(queryBody()).not.toBeNull();
  });

  it("Should keep the body open when interacting inside it so text selection stays safe", () => {
    render(<ToolCallRow message={makeToolMessage({ toolResult: { content: "abc" } })} />);
    const trigger = screen.getByRole("button");
    fireEvent.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(queryBody()).not.toBeNull();

    const input = document.querySelector<HTMLElement>('[data-slot="tool-call-row-input"]');
    expect(input).not.toBeNull();
    fireEvent.pointerDown(input as HTMLElement);
    fireEvent.click(input as HTMLElement);

    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(queryBody()).not.toBeNull();
  });

  it("Should render the existing Output dispatcher inside the inline body", () => {
    render(
      <ToolCallRow
        message={makeToolMessage({
          toolName: "Bash",
          toolInput: { command: "printf abc" },
          toolResult: { stdout: "abc" },
        })}
      />
    );
    const trigger = screen.getByRole("button");
    fireEvent.click(trigger);

    expect(document.querySelector('[data-slot="tool-call-row-output"]')).not.toBeNull();
    expect(queryBody()).toHaveTextContent("abc");
  });

  it("Should preserve the Edit output renderer inside the inline body", () => {
    render(
      <ToolCallRow
        message={makeToolMessage({
          toolName: "Edit",
          toolInput: {
            file_path: "/src/app.ts",
            old_string: "const oldValue = true;",
            new_string: "const oldValue = false;",
          },
          toolResult: { content: "Applied patch successfully." },
        })}
      />
    );

    fireEvent.click(screen.getByRole("button"));

    expect(screen.getByTestId("edit-content")).toBeInTheDocument();
    expect(queryBody()).toHaveTextContent("/src/app.ts");
    expect(queryBody()).toHaveTextContent("const oldValue = true;");
    expect(queryBody()).toHaveTextContent("const oldValue = false;");
  });

  it("Should keep MCP and dynamic tools on the generic output renderer", () => {
    render(
      <ToolCallRow
        message={makeToolMessage({
          toolName: "mcp__context7__resolve-library-id",
          toolInput: { libraryName: "react" },
          toolResult: { content: "/websites/react_dev" },
        })}
      />
    );

    fireEvent.click(screen.getByRole("button"));

    expect(queryToolName()).toHaveTextContent("mcp__context7__resolve-library-id");
    expect(queryBody()).toHaveTextContent('"libraryName": "react"');
    expect(queryBody()).toHaveTextContent("/websites/react_dev");
  });

  it("Should keep Output absent while the tool is still running", () => {
    render(<ToolCallRow message={makeToolMessage()} />);
    fireEvent.click(screen.getByRole("button"));

    expect(document.querySelector('[data-slot="tool-call-row-output"]')).toBeNull();
  });

  it("Should copy the structured tool payload from the expanded body", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    render(<ToolCallRow message={makeToolMessage({ toolResult: { content: "abc" } })} />);

    fireEvent.click(screen.getByRole("button"));
    fireEvent.click(screen.getByRole("button", { name: "Copy tool payload" }));

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const payload = JSON.parse(writeText.mock.calls[0]?.[0] as string) as {
      tool: string;
      input: Record<string, unknown>;
      output: Record<string, unknown>;
    };
    expect(payload.tool).toBe("Read");
    expect(payload.input.file_path).toBe("/src/main.ts");
    expect(payload.output.content).toBe("abc");
  });
});
