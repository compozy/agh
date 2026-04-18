import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Pill, Pills } from "./pills";

describe("Pill", () => {
  it("Should render a semantic tag with the success tint token as background", () => {
    render(<Pill variant="success">Live</Pill>);
    const pill = screen.getByText("Live");
    expect(pill).toHaveAttribute("data-slot", "pill");
    expect(pill).toHaveAttribute("data-variant", "success");
    expect(pill.className).toContain("bg-[color:var(--color-success-tint)]");
    expect(pill.className).toContain("text-[color:var(--color-success)]");
  });

  it("Should fall back to the default variant when none is provided", () => {
    render(<Pill>Neutral</Pill>);
    const pill = screen.getByText("Neutral");
    expect(pill).toHaveAttribute("data-variant", "default");
    expect(pill.className).toContain("border-[color:var(--color-divider)]");
  });

  it("Should use the md size token when size='md' is requested", () => {
    render(<Pill size="md">Filter</Pill>);
    const pill = screen.getByText("Filter");
    expect(pill).toHaveAttribute("data-size", "md");
    expect(pill.className).toContain("h-8");
  });
});

describe("Pills", () => {
  const items = [
    { value: "list", label: "List" },
    { value: "kanban", label: "Kanban" },
    { value: "inbox", label: "Inbox", badge: 3 },
  ] as const;

  it("Should fire onChange with the selected value when a tab is clicked", async () => {
    const user = userEvent.setup();
    const handle = vi.fn();
    render(<Pills value="list" onChange={handle} items={items} />);

    await user.click(screen.getByRole("tab", { name: /kanban/i }));

    expect(handle).toHaveBeenCalledWith("kanban");
  });

  it("Should reflect the active item via aria-selected + data-active", () => {
    render(<Pills value="kanban" onChange={() => {}} items={items} />);
    const kanban = screen.getByRole("tab", { name: /kanban/i });
    const list = screen.getByRole("tab", { name: /list/i });
    expect(kanban).toHaveAttribute("aria-selected", "true");
    expect(kanban).toHaveAttribute("data-active", "true");
    expect(list).toHaveAttribute("aria-selected", "false");
    expect(list).toHaveAttribute("data-active", "false");
  });

  it("Should render the badge count next to the item label when badge > 0", () => {
    render(<Pills value="list" onChange={() => {}} items={items} />);
    const inbox = screen.getByRole("tab", { name: /inbox/i });
    const badge = inbox.querySelector('[data-slot="pills-badge"]');
    expect(badge).not.toBeNull();
    expect(badge?.textContent).toBe("3");
  });

  it("Should not fire onChange for a disabled item", async () => {
    const user = userEvent.setup();
    const handle = vi.fn();
    render(
      <Pills
        value="list"
        onChange={handle}
        items={[
          { value: "list", label: "List" },
          { value: "kanban", label: "Kanban", disabled: true },
        ]}
      />
    );

    const kanban = screen.getByRole("tab", { name: /kanban/i });
    expect(kanban).toBeDisabled();

    await user.click(kanban);
    expect(handle).not.toHaveBeenCalled();
  });

  it("Should expose testId as data-testid when provided", () => {
    render(
      <Pills
        value="list"
        onChange={() => {}}
        items={[{ value: "list", label: "List", testId: "mode-list" }]}
      />
    );
    expect(screen.getByRole("tab", { name: /list/i })).toHaveAttribute("data-testid", "mode-list");
  });
});
