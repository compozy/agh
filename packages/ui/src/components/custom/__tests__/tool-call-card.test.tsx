import { fireEvent, render } from "@testing-library/react";
import { FileEditIcon } from "lucide-react";
import { describe, expect, it } from "vitest";

import { TOOL_CALL_STATUS_LABEL, ToolCallCard, type ToolCallStatus } from "../tool-call-card";

const TOOL_NAME = "fs.read_file";
const STATUSES: ToolCallStatus[] = ["pending", "in_progress", "completed", "failed"];

function makeLines(count: number): string {
  return Array.from({ length: count }, (_, index) => `line-${index}`).join("\n");
}

function queryInputToggle(container: ParentNode): HTMLButtonElement | null {
  return container.querySelector<HTMLButtonElement>('[data-slot="tool-call-card-input-toggle"]');
}

function queryOutputToggle(container: ParentNode): HTMLButtonElement | null {
  return container.querySelector<HTMLButtonElement>('[data-slot="tool-call-card-output-toggle"]');
}

describe("ToolCallCard", () => {
  it("Should render the terminal icon, tool name, and file path in the header", () => {
    const { container } = render(
      <ToolCallCard
        toolName="file.read"
        filePath="packages/runtime/src/session/stream.ts"
        status="in_progress"
      />
    );
    const icon = container.querySelector<SVGElement>('[data-slot="tool-call-card-icon"]');
    const tool = container.querySelector<HTMLElement>('[data-slot="tool-call-card-tool"]');
    const path = container.querySelector<HTMLElement>('[data-slot="tool-call-card-path"]');
    expect(icon).not.toBeNull();
    expect((icon as unknown as SVGElement).classList.contains("lucide-terminal")).toBe(true);
    expect(tool?.textContent).toBe("file.read");
    expect(path?.textContent).toBe("packages/runtime/src/session/stream.ts");
  });

  it("Should omit the file path slot when filePath is undefined", () => {
    const { container } = render(<ToolCallCard toolName="shell.run" status="in_progress" />);
    expect(container.querySelector('[data-slot="tool-call-card-path"]')).toBeNull();
  });

  it("Should map every status to its expected icon + aria-label", () => {
    expect(TOOL_CALL_STATUS_LABEL).toEqual({
      pending: "Pending",
      in_progress: "Running",
      completed: "Done",
      failed: "Error",
    });
    for (const status of STATUSES) {
      const { container, unmount } = render(<ToolCallCard toolName={TOOL_NAME} status={status} />);
      const root = container.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
      const indicator = container.querySelector<HTMLElement>('[data-slot="tool-call-card-status"]');
      expect(root?.getAttribute("data-status")).toBe(status);
      expect(indicator?.getAttribute("data-status")).toBe(status);
      expect(indicator?.getAttribute("aria-label")).toBe(TOOL_CALL_STATUS_LABEL[status]);
      unmount();
    }
  });

  it("Should render a spinner with role=status for in_progress", () => {
    const { container } = render(<ToolCallCard toolName={TOOL_NAME} status="in_progress" />);
    const indicator = container.querySelector<SVGElement>('[data-slot="tool-call-card-status"]');
    expect(indicator?.getAttribute("role")).toBe("status");
    expect(indicator?.classList.contains("animate-spin")).toBe(true);
  });

  it("Should render the optional actions slot in the header", () => {
    const { container } = render(
      <ToolCallCard
        toolName={TOOL_NAME}
        status="in_progress"
        actions={<button type="button">Retry</button>}
      />
    );
    const actions = container.querySelector<HTMLElement>('[data-slot="tool-call-card-actions"]');
    expect(actions?.textContent).toBe("Retry");
  });

  it("Should paint the failure ring + render the error message slot when failed", () => {
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="failed" errorMessage="ENOENT: no such file" />
    );
    const root = container.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
    expect(root?.getAttribute("data-status")).toBe("failed");
    const error = container.querySelector<HTMLElement>('[data-slot="tool-call-card-error"]');
    expect(error?.textContent).toBe("ENOENT: no such file");
  });

  it("Should mark the body as empty when Input/Output exist but all chips are closed", () => {
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="completed">
        <ToolCallCard.Input source="argument" format="code" />
        <ToolCallCard.Output source="result" format="code" />
      </ToolCallCard>
    );
    const body = container.querySelector<HTMLElement>('[data-slot="tool-call-card-body"]');
    expect(body).not.toBeNull();
    expect(body?.getAttribute("data-empty")).toBe("true");
    expect(body?.className).not.toContain("border-t");
    expect(container.querySelector('[data-slot="tool-call-card-chip-group"]')).not.toBeNull();
  });

  it("Should render disclosure chips in the header, not in the body", () => {
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="in_progress">
        <ToolCallCard.Input source="argument" format="code" />
      </ToolCallCard>
    );
    const header = container.querySelector('[data-slot="tool-call-card-header"]');
    const toggle = queryInputToggle(container);
    expect(header?.contains(toggle)).toBe(true);
    const body = container.querySelector<HTMLElement>('[data-slot="tool-call-card-body"]');
    expect(body?.getAttribute("data-empty")).toBe("true");
    expect(body?.className).not.toContain("border-t");
  });

  it("Should render the Input chip closed by default and toggle open on click", () => {
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="in_progress">
        <ToolCallCard.Input source="argument" format="code" />
      </ToolCallCard>
    );
    const toggle = queryInputToggle(container);
    expect(toggle?.getAttribute("aria-expanded")).toBe("false");
    expect(toggle?.getAttribute("data-open")).toBe("false");
    expect(container.querySelector('[data-slot="tool-call-card-input-body"]')).toBeNull();
    fireEvent.click(toggle!);
    const section = container.querySelector<HTMLElement>('[data-slot="tool-call-card-input"]');
    expect(section?.getAttribute("data-open")).toBe("true");
    expect(container.querySelector('[data-slot="tool-call-card-input-body"]')).not.toBeNull();
    const body = container.querySelector<HTMLElement>('[data-slot="tool-call-card-body"]');
    expect(body?.className).toContain("border-t");
  });

  it("Should render the Output chip closed by default even when content is long", () => {
    const longOutput = makeLines(300);
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="completed">
        <ToolCallCard.Output source={longOutput} format="code" />
      </ToolCallCard>
    );
    const toggle = queryOutputToggle(container);
    expect(toggle?.getAttribute("aria-expanded")).toBe("false");
    expect(container.querySelector('[data-slot="tool-call-card-output-body"]')).toBeNull();
    fireEvent.click(toggle!);
    expect(
      container.querySelector('[data-slot="tool-call-card-output"]')?.getAttribute("data-open")
    ).toBe("true");
    expect(container.querySelector('[data-slot="tool-call-card-output-body"]')).not.toBeNull();
  });

  it("Should honor defaultOpen on Input + Output sections", () => {
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="completed">
        <ToolCallCard.Input source="argument" format="code" defaultOpen />
        <ToolCallCard.Output source="output" format="code" defaultOpen />
      </ToolCallCard>
    );
    expect(
      container.querySelector('[data-slot="tool-call-card-input"]')?.getAttribute("data-open")
    ).toBe("true");
    expect(
      container.querySelector('[data-slot="tool-call-card-output"]')?.getAttribute("data-open")
    ).toBe("true");
  });

  it("Should prefer `children` over `source` when both are provided", () => {
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="completed">
        <ToolCallCard.Output source="ignored" format="code" defaultOpen>
          <span data-testid="custom-output">custom</span>
        </ToolCallCard.Output>
      </ToolCallCard>
    );
    expect(container.querySelector('[data-testid="custom-output"]')?.textContent).toBe("custom");
    expect(container.querySelector('[data-slot="tool-call-card-output-body"] pre')).toBeNull();
  });

  it("Should render markdown sources through the canonical Markdown primitive", () => {
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="in_progress">
        <ToolCallCard.Input
          defaultOpen
          source={"<script>alert(1)</script>\n\n**arg**"}
          format="markdown"
        />
      </ToolCallCard>
    );
    const body = container.querySelector('[data-slot="tool-call-card-input-body"]');
    const markdown = body?.querySelector('[data-slot="markdown"]');
    expect(markdown).not.toBeNull();
    expect(markdown?.getAttribute("data-compact")).toBe("true");
    expect(body?.querySelector("script")).toBeNull();
    expect(body?.querySelector("strong")?.textContent).toBe("arg");
    expect(body?.textContent ?? "").not.toContain("alert(1)");
  });

  it("Should still accept raw children as body content (stdout, diffs, …)", () => {
    const { container } = render(
      <ToolCallCard toolName={TOOL_NAME} status="completed">
        <pre data-testid="raw-stdout">$ ls</pre>
      </ToolCallCard>
    );
    expect(container.querySelector('[data-testid="raw-stdout"]')?.textContent).toBe("$ ls");
    const body = container.querySelector<HTMLElement>('[data-slot="tool-call-card-body"]');
    expect(body).not.toBeNull();
    expect(body?.getAttribute("data-empty")).toBeNull();
    expect(body?.className).toContain("border-t");
  });

  it("Should mark the body as empty when no body content is provided", () => {
    const { container } = render(<ToolCallCard toolName={TOOL_NAME} status="pending" />);
    const body = container.querySelector<HTMLElement>('[data-slot="tool-call-card-body"]');
    expect(body).not.toBeNull();
    expect(body?.getAttribute("data-empty")).toBe("true");
    expect(body?.className).not.toContain("border-t");
  });

  it("Should forward className and extra props to the root container", () => {
    const { container } = render(
      <ToolCallCard toolName="t" status="in_progress" className="ring-1" data-testid="card" />
    );
    const root = container.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
    expect(root?.className).toContain("ring-1");
    expect(root?.getAttribute("data-testid")).toBe("card");
  });

  it("Should render a custom Lucide icon component when icon prop is a component ref", () => {
    const { container } = render(
      <ToolCallCard toolName="file.edit" status="completed" icon={FileEditIcon} />
    );
    const icon = container.querySelector<SVGElement>('[data-slot="tool-call-card-icon"]');
    expect(icon).not.toBeNull();
    expect((icon as unknown as SVGElement).classList.contains("lucide-file-pen")).toBe(true);
  });

  it("Should render a pre-rendered ReactNode icon as-is", () => {
    const { container } = render(
      <ToolCallCard
        toolName="file.edit"
        status="completed"
        icon={<span data-testid="custom-icon">~</span>}
      />
    );
    expect(container.querySelector('[data-testid="custom-icon"]')?.textContent).toBe("~");
  });
});
