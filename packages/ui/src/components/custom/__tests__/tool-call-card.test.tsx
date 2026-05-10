import { render } from "@testing-library/react";
import { FileEditIcon } from "lucide-react";
import { describe, expect, it } from "vitest";

import { ToolCallCard, type ToolCallStatus } from "../tool-call-card";

const TONE_BY_STATUS: Record<ToolCallStatus, { tone: string; label: string }> = {
  running: { tone: "accent", label: "RUNNING" },
  done: { tone: "success", label: "DONE" },
  error: { tone: "danger", label: "ERROR" },
};

describe("ToolCallCard", () => {
  it("Should render the terminal icon, tool name, and file path in the header", () => {
    const { container } = render(
      <ToolCallCard
        toolName="file.read"
        filePath="packages/runtime/src/session/stream.ts"
        status="running"
      />
    );
    const root = container.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
    const icon = container.querySelector<SVGElement>('[data-slot="tool-call-card-icon"]');
    const tool = container.querySelector<HTMLElement>('[data-slot="tool-call-card-tool"]');
    const path = container.querySelector<HTMLElement>('[data-slot="tool-call-card-path"]');
    expect(root?.className).toContain("bg-(--canvas-soft)");
    expect(root?.className).toContain("border-(--line)");
    expect(root?.className).toContain("rounded-md");
    expect(icon).not.toBeNull();
    expect((icon as unknown as SVGElement).classList.contains("lucide-terminal")).toBe(true);
    expect(tool?.textContent).toBe("file.read");
    expect(tool?.className).toContain("font-medium");
    expect(tool?.className).toContain("text-[14px]");
    expect(path?.textContent).toBe("packages/runtime/src/session/stream.ts");
    expect(path?.className).toContain("text-[13px]");
    expect(path?.className).toContain("text-(--subtle)");
  });

  it("Should omit the file path slot when filePath is undefined", () => {
    const { container } = render(<ToolCallCard toolName="shell.run" status="running" />);
    expect(container.querySelector('[data-slot="tool-call-card-path"]')).toBeNull();
  });

  it.each<ToolCallStatus>(["running", "done", "error"])(
    "Should render the status badge with the correct label and semantic tone for %s",
    status => {
      const { container } = render(<ToolCallCard toolName="t" status={status} />);
      const root = container.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
      const badge = container.querySelector<HTMLElement>('[data-slot="tool-call-card-status"]');
      expect(root?.getAttribute("data-status")).toBe(status);
      expect(badge?.textContent).toBe(TONE_BY_STATUS[status].label);
      expect(badge?.getAttribute("data-tone")).toBe(TONE_BY_STATUS[status].tone);
      expect(badge?.className).toContain("ml-auto");
    }
  );

  it("Should render optional children in a bordered body slot below the header", () => {
    const { container } = render(
      <ToolCallCard toolName="file.read" status="done">
        <pre data-testid="output">$ cat file.txt</pre>
      </ToolCallCard>
    );
    const body = container.querySelector<HTMLElement>('[data-slot="tool-call-card-body"]');
    const header = container.querySelector<HTMLElement>('[data-slot="tool-call-card-header"]');
    expect(body).not.toBeNull();
    expect(body?.className).toContain("border-t");
    expect(body?.className).toContain("border-(--line)");
    expect(body?.querySelector('[data-testid="output"]')?.textContent).toBe("$ cat file.txt");
    const bodyIndex = Array.prototype.indexOf.call(body?.parentElement?.children ?? [], body);
    const headerIndex = Array.prototype.indexOf.call(header?.parentElement?.children ?? [], header);
    expect(bodyIndex).toBeGreaterThan(headerIndex);
  });

  it("Should not render the body slot when children is omitted", () => {
    const { container } = render(<ToolCallCard toolName="file.read" status="done" />);
    expect(container.querySelector('[data-slot="tool-call-card-body"]')).toBeNull();
  });

  it("Should forward className and extra props to the root container", () => {
    const { container } = render(
      <ToolCallCard toolName="t" status="running" className="ring-1" data-testid="card" />
    );
    const root = container.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
    expect(root?.className).toContain("ring-1");
    expect(root?.getAttribute("data-testid")).toBe("card");
  });

  it("Should render a custom Lucide icon component when icon prop is a component ref", () => {
    const { container } = render(
      <ToolCallCard toolName="file.edit" status="done" icon={FileEditIcon} />
    );
    const icon = container.querySelector<SVGElement>('[data-slot="tool-call-card-icon"]');
    expect(icon).not.toBeNull();
    expect((icon as unknown as SVGElement).classList.contains("lucide-file-pen")).toBe(true);
  });

  it("Should render a pre-rendered ReactNode icon as-is", () => {
    const { container } = render(
      <ToolCallCard
        toolName="file.edit"
        status="done"
        icon={<span data-testid="custom-icon">✎</span>}
      />
    );
    expect(container.querySelector('[data-testid="custom-icon"]')?.textContent).toBe("✎");
  });

  it("Should apply a danger-toned border when status is error", () => {
    const { container } = render(<ToolCallCard toolName="t" status="error" />);
    const root = container.querySelector<HTMLElement>('[data-slot="tool-call-card"]');
    expect(root?.className).toContain("data-[status=error]:border-(--danger)/40");
  });
});
