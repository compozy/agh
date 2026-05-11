import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DetailHeader } from "../detail-header";

describe("DetailHeader", () => {
  it("Should render the 6-row stack in order: crumbs → preTitle → title → pills → meta → actions", () => {
    const { container } = render(
      <DetailHeader
        crumbs="Tasks / detail"
        preTitle="Run #42"
        title="Refactor"
        pills={<span data-testid="pill">Active</span>}
        meta={<span>id-42</span>}
        actions={<button type="button">Run</button>}
      />
    );
    const slots = Array.from(container.querySelectorAll<HTMLElement>("[data-slot]")).map(
      el => el.dataset.slot
    );
    const order = slots.filter(
      s =>
        s === "detail-header-crumbs" ||
        s === "detail-header-pre-title" ||
        s === "detail-header-title" ||
        s === "detail-header-pills" ||
        s === "detail-header-meta" ||
        s === "detail-header-actions"
    );
    expect(order).toEqual([
      "detail-header-crumbs",
      "detail-header-pre-title",
      "detail-header-title",
      "detail-header-pills",
      "detail-header-meta",
      "detail-header-actions",
    ]);
  });

  it("Should invoke the back callback when the back affordance is clicked", () => {
    const back = vi.fn();
    render(<DetailHeader title="Untitled" crumbs="Tasks" back={back} />);
    const button = screen.getByRole("button", { name: /go back/i });
    fireEvent.click(button);
    expect(back).toHaveBeenCalledTimes(1);
  });

  it("Should render structured crumbs with `·` separators", () => {
    render(
      <DetailHeader
        title="Run"
        crumbs={[{ label: "Workspaces" }, { label: "Sessions" }, { label: "Run #42" }]}
      />
    );
    const crumbList = screen
      .getByText("Workspaces")
      .closest('[data-slot="detail-header-crumbs-list"]');
    expect(crumbList).not.toBeNull();
    const separators = crumbList?.querySelectorAll('[aria-hidden="true"]');
    expect(separators?.length).toBe(2);
  });

  it("Should render crumbs without the back button when `back` is not provided", () => {
    render(<DetailHeader title="Untitled" crumbs="Tasks" />);
    expect(screen.queryByRole("button", { name: /go back/i })).toBeNull();
  });
});
