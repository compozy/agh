import { act, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CodeBlock } from "../code-block";

function installClipboard() {
  const writeText = vi.fn<(value: string) => Promise<void>>().mockResolvedValue(undefined);
  const descriptor = Object.getOwnPropertyDescriptor(navigator, "clipboard");
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
  const restore = () => {
    if (descriptor) {
      Object.defineProperty(navigator, "clipboard", descriptor);
    } else {
      Reflect.deleteProperty(navigator as unknown as { clipboard?: unknown }, "clipboard");
    }
  };
  return { writeText, restore };
}

describe("CodeBlock", () => {
  let clipboard: ReturnType<typeof installClipboard>;

  beforeEach(() => {
    clipboard = installClipboard();
  });

  afterEach(() => {
    clipboard.restore();
    vi.useRealTimers();
  });

  it("Should render the provided code inside a <pre><code> wrapper using JetBrains Mono", () => {
    const { container } = render(<CodeBlock code="agh start" />);
    const root = container.querySelector<HTMLElement>('[data-slot="code-block"]');
    const pre = container.querySelector<HTMLElement>('[data-slot="code-block-pre"]');
    const code = container.querySelector<HTMLElement>('[data-slot="code-block-code"]');
    expect(root?.className).toContain("bg-[color:var(--color-canvas-deep)]");
    expect(root?.className).toContain("rounded-[var(--radius-diagram)]");
    expect(pre?.tagName).toBe("PRE");
    expect(code?.tagName).toBe("CODE");
    expect(pre?.className).toContain("font-mono");
    expect(pre?.className).toContain("text-[14px]");
    expect(pre?.className).toContain("leading-[1.6]");
    expect(code?.textContent).toContain("agh start");
  });

  it("Should color the `$ ` prompt in accent when showPrompt is true", () => {
    const { container } = render(<CodeBlock code="agh start" />);
    const prompt = container.querySelector<HTMLElement>('[data-slot="code-block-prompt"]');
    expect(prompt).not.toBeNull();
    expect(prompt?.textContent).toBe("$ ");
    expect(prompt?.className).toContain("text-[color:var(--color-accent)]");
    expect(prompt?.getAttribute("aria-hidden")).toBe("true");
  });

  it("Should suppress the prompt on continuation and comment lines", () => {
    const code = ["# comment", "agh network status", '    --body \'{"task":"go"}\'', ""].join("\n");
    const { container } = render(<CodeBlock code={code} />);
    const prompts = container.querySelectorAll('[data-slot="code-block-prompt"]');
    expect(prompts.length).toBe(1);
  });

  it("Should omit prompts entirely when showPrompt is false", () => {
    const { container } = render(
      <CodeBlock code={"const x = 1;\nconst y = 2;"} showPrompt={false} />
    );
    expect(container.querySelector('[data-slot="code-block-prompt"]')).toBeNull();
  });

  it("Should render the language eyebrow only when the language prop is provided", () => {
    const { container, rerender } = render(<CodeBlock code="agh start" />);
    expect(container.querySelector('[data-slot="code-block-language"]')).toBeNull();
    rerender(<CodeBlock code="agh start" language="shell" />);
    const eyebrow = container.querySelector<HTMLElement>('[data-slot="code-block-language"]');
    expect(eyebrow?.textContent).toBe("shell");
    expect(eyebrow?.className).toContain("uppercase");
    expect(eyebrow?.className).toContain("font-mono");
    expect(eyebrow?.className).toContain("tracking-[0.06em]");
  });

  it("Should hide the copy button when copyable is false", () => {
    const { container } = render(<CodeBlock code="agh start" copyable={false} />);
    expect(container.querySelector('[data-slot="code-block-copy"]')).toBeNull();
  });

  it("Should call navigator.clipboard.writeText with the code when copy is clicked", async () => {
    const { container } = render(<CodeBlock code="agh network status" />);
    const button = container.querySelector<HTMLButtonElement>('[data-slot="code-block-copy"]')!;
    fireEvent.click(button);
    await waitFor(() => {
      expect(clipboard.writeText).toHaveBeenCalledWith("agh network status");
    });
  });

  it("Should swap to the check icon for 1.5s on copy success, then revert", async () => {
    vi.useFakeTimers();
    const { container } = render(<CodeBlock code="agh start" />);
    const button = container.querySelector<HTMLButtonElement>('[data-slot="code-block-copy"]')!;
    expect(button.querySelector("svg.lucide-copy")).not.toBeNull();
    expect(button.querySelector("svg.lucide-check")).toBeNull();
    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(button.getAttribute("data-copied")).toBe("true");
    expect(button.querySelector("svg.lucide-check")).not.toBeNull();
    expect(button.querySelector("svg.lucide-copy")).toBeNull();
    await act(async () => {
      vi.advanceTimersByTime(1500);
    });
    expect(button.getAttribute("data-copied")).toBeNull();
    expect(button.querySelector("svg.lucide-copy")).not.toBeNull();
  });

  it("Should restart the copy feedback timer when copy is clicked repeatedly", async () => {
    vi.useFakeTimers();
    const { container } = render(<CodeBlock code="agh start" />);
    const button = container.querySelector<HTMLButtonElement>('[data-slot="code-block-copy"]')!;

    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(button.getAttribute("data-copied")).toBe("true");

    await act(async () => {
      vi.advanceTimersByTime(1000);
    });

    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
      await Promise.resolve();
    });

    await act(async () => {
      vi.advanceTimersByTime(1499);
    });
    expect(button.getAttribute("data-copied")).toBe("true");

    await act(async () => {
      vi.advanceTimersByTime(1);
    });
    expect(button.getAttribute("data-copied")).toBeNull();
  });
});
