import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { PageActionsTopbarSlot } from "../page-actions-topbar-slot";

describe("PageActionsTopbarSlot", () => {
  it("Should render save + discard active when dirty is true and call back on click", () => {
    const onSave = vi.fn();
    const onDiscard = vi.fn();
    const { container } = render(
      <PageActionsTopbarSlot dirty onSave={onSave} onDiscard={onDiscard} />
    );

    const root = container.querySelector<HTMLElement>('[data-slot="page-actions-topbar-slot"]');
    expect(root?.dataset.dirty).toBe("true");

    const save = screen.getByRole("button", { name: "Save changes" });
    const discard = screen.getByRole("button", { name: "Discard" });
    expect(save).toBeEnabled();
    expect(discard).toBeEnabled();

    fireEvent.click(save);
    fireEvent.click(discard);
    expect(onSave).toHaveBeenCalledTimes(1);
    expect(onDiscard).toHaveBeenCalledTimes(1);
  });

  it("Should disable both buttons when dirty is false", () => {
    const { container } = render(
      <PageActionsTopbarSlot dirty={false} onSave={vi.fn()} onDiscard={vi.fn()} />
    );
    expect(screen.getByRole("button", { name: "Save changes" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Discard" })).toBeDisabled();
    const root = container.querySelector<HTMLElement>('[data-slot="page-actions-topbar-slot"]');
    expect(root?.dataset.dirty).toBe("false");
  });

  it("Should swap the save label to Saving... and disable both buttons while saving", () => {
    const { container } = render(
      <PageActionsTopbarSlot dirty saving onSave={vi.fn()} onDiscard={vi.fn()} />
    );
    expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Discard" })).toBeDisabled();
    const root = container.querySelector<HTMLElement>('[data-slot="page-actions-topbar-slot"]');
    expect(root?.dataset.saving).toBe("true");
  });
});
