import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  CHAT_TOOL_OUTPUT_COLLAPSE_LINES,
  ChatToolCard,
  TOOL_STATUS_LABEL,
  TOOL_STATUS_TONE,
} from "../chat-tool-card";

const TOOL_NAME = "fs.read_file";

function makeLines(count: number): string {
  return Array.from({ length: count }, (_, index) => `line-${index}`).join("\n");
}

describe("ChatToolCard", () => {
  it("Should render the head row with tool name as <MonoId> + status pill + relative <Time>", () => {
    const { container } = render(
      <ChatToolCard toolName={TOOL_NAME} status="in_progress" timestamp="2026-05-11T12:00:00Z" />
    );
    const name = container.querySelector<HTMLElement>('[data-slot="chat-tool-card-name"]');
    expect(name?.dataset.slot).toBe("chat-tool-card-name");
    expect(name?.textContent).toContain(TOOL_NAME.toLowerCase());

    const status = container.querySelector<HTMLElement>('[data-slot="chat-tool-card-status"]');
    expect(status?.dataset.tone).toBe(TOOL_STATUS_TONE.in_progress);
    expect(status?.textContent).toBe(TOOL_STATUS_LABEL.in_progress);

    const time = container.querySelector<HTMLElement>('[data-slot="chat-tool-card-time"]');
    expect(time?.tagName).toBe("TIME");
    expect(time?.getAttribute("datetime")).toBe("2026-05-11T12:00:00Z");
  });

  it("Should map every ChatToolStatus to its expected PillTone", () => {
    expect(TOOL_STATUS_TONE).toEqual({
      pending: "neutral",
      in_progress: "info",
      completed: "success",
      failed: "danger",
    });
  });

  it("Should default output to collapsed when string length exceeds the 200-line threshold", () => {
    const longOutput = makeLines(CHAT_TOOL_OUTPUT_COLLAPSE_LINES + 1);
    const { container } = render(
      <ChatToolCard
        toolName={TOOL_NAME}
        status="completed"
        output={{ source: longOutput, format: "code" }}
      />
    );
    const section = container.querySelector<HTMLElement>('[data-slot="chat-tool-card-output"]');
    expect(section?.dataset.open).toBe("false");
    expect(container.querySelector('[data-slot="chat-tool-card-output-body"]')).toBeNull();
  });

  it("Should keep output expanded when content stays under the threshold", () => {
    const shortOutput = makeLines(10);
    const { container } = render(
      <ChatToolCard
        toolName={TOOL_NAME}
        status="completed"
        output={{ source: shortOutput, format: "code" }}
      />
    );
    const section = container.querySelector<HTMLElement>('[data-slot="chat-tool-card-output"]');
    expect(section?.dataset.open).toBe("true");
    expect(container.querySelector('[data-slot="chat-tool-card-output-body"]')).not.toBeNull();
  });

  it("Should toggle the input region open/closed on click", () => {
    const { container } = render(
      <ChatToolCard
        toolName={TOOL_NAME}
        status="in_progress"
        input={{ source: "argument", format: "code" }}
        initialInputCollapsed
      />
    );
    const section = container.querySelector<HTMLElement>('[data-slot="chat-tool-card-input"]');
    expect(section?.dataset.open).toBe("false");
    const toggle = container.querySelector<HTMLButtonElement>(
      '[data-slot="chat-tool-card-input-toggle"]'
    );
    fireEvent.click(toggle!);
    expect(section?.dataset.open).toBe("true");
  });

  it("Should emit an error message in failed state", () => {
    const { container } = render(
      <ChatToolCard toolName={TOOL_NAME} status="failed" errorMessage="ENOENT: no such file" />
    );
    const root = container.querySelector<HTMLElement>('[data-slot="chat-tool-card"]');
    expect(root?.dataset.status).toBe("failed");
    const error = container.querySelector<HTMLElement>('[data-slot="chat-tool-card-error"]');
    expect(error?.textContent).toBe("ENOENT: no such file");
  });

  it("Should render markdown input through the Streamdown safe contract", () => {
    const { container } = render(
      <ChatToolCard
        toolName={TOOL_NAME}
        status="in_progress"
        input={{ source: "<script>alert(1)</script>\n\n**arg**", format: "markdown" }}
      />
    );
    const body = container.querySelector('[data-slot="chat-tool-card-input-body"]');
    expect(body?.querySelector("script")).toBeNull();
    expect(body?.querySelector("strong")?.textContent).toBe("arg");
    expect(body?.textContent ?? "").not.toContain("alert(1)");
  });

  it("Should render trailing actions slot when provided", () => {
    const { container } = render(
      <ChatToolCard
        toolName={TOOL_NAME}
        status="failed"
        actions={<button type="button">Retry</button>}
      />
    );
    const actions = container.querySelector<HTMLElement>('[data-slot="chat-tool-card-actions"]');
    expect(actions?.textContent).toBe("Retry");
  });
});
