import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { MonoId } from "../mono-id";

describe("MonoId", () => {
  it("Should render bare lowercase mono identifier at --text-mono-id / --faint", () => {
    const { container } = render(<MonoId value="run_ABC123" />);
    const root = container.querySelector<HTMLElement>('[data-slot="mono-id"]');
    expect(root?.dataset.size).toBe("default");
    expect(root?.className).toContain("font-mono");
    expect(root?.className).toContain("text-[length:var(--text-mono-id)]");
    expect(root?.className).toContain("tracking-(--tracking-mono-id)");
    expect(root?.className).toContain("text-(--faint)");
    expect(root?.className).not.toContain("rounded");
    expect(root?.className).not.toContain("bg-(--neutral-tint)");

    const value = container.querySelector<HTMLElement>('[data-slot="mono-id-value"]');
    expect(value?.textContent).toBe("run_abc123");
  });

  it("Should switch to 10 px when size is sm", () => {
    const { container } = render(<MonoId value="run_abc" size="sm" />);
    const root = container.querySelector<HTMLElement>('[data-slot="mono-id"]');
    expect(root?.dataset.size).toBe("sm");
    expect(root?.className).toContain("text-[10px]");
  });

  it("Should render an inline copy button only when copy is true", () => {
    const { rerender } = render(<MonoId value="run_abc" />);
    expect(screen.queryByRole("button", { name: /copy/i })).toBeNull();

    rerender(<MonoId value="run_abc" copy />);
    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();
  });

  it("Should write the lowercased value to the clipboard when copy is pressed", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<MonoId value="RUN_ABC" copy />);
    fireEvent.click(screen.getByRole("button", { name: /copy/i }));
    expect(writeText).toHaveBeenCalledWith("run_abc");
  });
});
