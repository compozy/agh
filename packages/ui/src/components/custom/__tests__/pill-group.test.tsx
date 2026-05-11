import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { PillGroup } from "../pill-group";

describe("PillGroup", () => {
  const items = [
    { value: "list", label: "List" },
    { value: "kanban", label: "Kanban" },
    { value: "inbox", label: "Inbox", badge: 3 },
  ] as const;

  it("Should fire onChange with the selected value when a non-active item is clicked", async () => {
    const user = userEvent.setup();
    const handle = vi.fn();
    render(<PillGroup value="list" onChange={handle} items={items} />);

    await user.click(screen.getByRole("button", { name: /kanban/i }));

    expect(handle).toHaveBeenCalledWith("kanban");
  });

  it("Should not fire onChange when the active item is re-clicked", async () => {
    const user = userEvent.setup();
    const handle = vi.fn();
    render(<PillGroup value="list" onChange={handle} items={items} />);

    await user.click(screen.getByRole("button", { name: /list/i }));

    expect(handle).not.toHaveBeenCalled();
  });

  it("Should reflect the active item via aria-pressed and data-active", () => {
    render(<PillGroup value="kanban" onChange={() => {}} items={items} />);
    const kanban = screen.getByRole("button", { name: /kanban/i });
    const list = screen.getByRole("button", { name: /list/i });
    expect(kanban).toHaveAttribute("aria-pressed", "true");
    expect(kanban).toHaveAttribute("data-active", "true");
    expect(list).toHaveAttribute("aria-pressed", "false");
    expect(list).toHaveAttribute("data-active", "false");
  });

  it("Should render the active segment with elevated background plus the highlight box-shadow", () => {
    render(<PillGroup value="kanban" onChange={() => {}} items={items} />);
    const kanban = screen.getByRole("button", { name: /kanban/i });
    expect(kanban.className).toContain("bg-(--elevated)");
    expect(kanban.className).toContain("text-(--fg-strong)");
    expect(kanban.className).toContain("shadow-(--highlight)");
  });

  it("Should render segment text as Inter sentence-case 12px / 510 / -0.005em (no font-mono, no uppercase)", () => {
    render(<PillGroup value="list" onChange={() => {}} items={items} />);
    const segments = screen
      .getAllByRole("button")
      .filter(node => node.dataset.slot === "pill-group-item");
    expect(segments).not.toHaveLength(0);
    for (const seg of segments) {
      expect(seg.className).toContain("text-[12px]");
      expect(seg.className).toContain("font-[510]");
      expect(seg.className).toContain("tracking-[-0.005em]");
      expect(seg.className).not.toContain("font-mono");
      expect(seg.className).not.toContain("uppercase");
      expect(seg.className).not.toContain("tracking-(--tracking-badge)");
    }
  });

  it("Should render the track without a border, with --canvas-soft fill, --radius-md corners, 2px padding and 1px gap", () => {
    const { container } = render(<PillGroup value="list" onChange={() => {}} items={items} />);
    const track = container.querySelector('[data-slot="pill-group"]') as HTMLElement | null;
    expect(track).not.toBeNull();
    expect(track?.className).toContain("bg-(--canvas-soft)");
    expect(track?.className).toContain("rounded-(--radius-md)");
    expect(track?.className).toContain("p-(--space-pill-group-track-padding)");
    expect(track?.className).toContain("gap-(--space-pill-group-track-gap)");
    expect(track?.className).not.toContain("border");
  });

  it("Should render the count badge as a 3px-radius neutral chip with tabular-nums and sentence-case", () => {
    render(<PillGroup value="list" onChange={() => {}} items={items} />);
    const inbox = screen.getByRole("button", { name: /inbox/i });
    const badge = inbox.querySelector('[data-slot="pill-group-badge"]') as HTMLElement | null;
    expect(badge).not.toBeNull();
    expect(badge?.textContent).toBe("3");
    expect(badge?.className).toContain("rounded-[3px]");
    expect(badge?.className).toContain("bg-(--badge-fill)");
    expect(badge?.className).toContain("text-(--muted)");
    expect(badge?.className).toContain("tabular-nums");
    expect(badge?.className).not.toContain("uppercase");
    expect(badge?.className).not.toContain("font-mono");
    expect(badge?.className).not.toContain("bg-(--accent)");
    expect(badge?.className).not.toContain("text-(--accent-ink)");
  });

  it("Should not fire onChange for a disabled item", async () => {
    const user = userEvent.setup();
    const handle = vi.fn();
    render(
      <PillGroup
        value="list"
        onChange={handle}
        items={[
          { value: "list", label: "List" },
          { value: "kanban", label: "Kanban", disabled: true },
        ]}
      />
    );

    const kanban = screen.getByRole("button", { name: /kanban/i });
    expect(kanban).toBeDisabled();

    await user.click(kanban);
    expect(handle).not.toHaveBeenCalled();
  });

  it("Should expose testId as data-testid when provided", () => {
    render(
      <PillGroup
        value="list"
        onChange={() => {}}
        items={[{ value: "list", label: "List", testId: "mode-list" }]}
      />
    );
    expect(screen.getByRole("button", { name: /list/i })).toHaveAttribute(
      "data-testid",
      "mode-list"
    );
  });

  it("Should render the larger md segments by default and switch to sm when requested", () => {
    const { container, rerender } = render(
      <PillGroup value="list" onChange={() => {}} items={items} />
    );
    let segments = container.querySelectorAll<HTMLElement>('[data-slot="pill-group-item"]');
    expect(segments[0]?.className).toContain("min-h-(--height-pill-group-segment-md)");

    rerender(<PillGroup value="list" onChange={() => {}} items={items} size="sm" />);
    segments = container.querySelectorAll<HTMLElement>('[data-slot="pill-group-item"]');
    expect(segments[0]?.className).toContain("min-h-(--height-pill-group-segment-sm)");
  });
});
