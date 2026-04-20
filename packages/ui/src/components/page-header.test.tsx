import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PageHeader } from "./page-header";

function DummyIcon({ className }: { className?: string }) {
  return <svg data-testid="page-header-icon-svg" className={className} />;
}

describe("PageHeader", () => {
  it("Should render title + optional icon + count badge + right-side meta in the mock order", () => {
    const { container } = render(
      <PageHeader
        title="Tasks"
        icon={DummyIcon}
        count={12}
        controls={<span data-testid="mode-pills">mode</span>}
        meta={<span data-testid="new-btn">new</span>}
      />
    );

    expect(screen.getByText("Tasks")).toBeInTheDocument();
    expect(screen.getByTestId("page-header-icon-svg")).toBeInTheDocument();

    const count = container.querySelector('[data-slot="page-header-count"]');
    expect(count).not.toBeNull();
    expect(count?.textContent).toBe("12");

    const header = container.querySelector('[data-slot="page-header"]');
    const slots = Array.from(header?.children ?? []).map(node => node.getAttribute("data-slot"));
    // mock order: title -> controls -> meta
    expect(slots).toEqual(["page-header-title", "page-header-controls", "page-header-meta"]);

    expect(screen.getByTestId("mode-pills")).toBeInTheDocument();
    expect(screen.getByTestId("new-btn")).toBeInTheDocument();
  });

  it("Should omit the icon slot when no icon is passed", () => {
    const { container } = render(<PageHeader title="Settings" />);
    expect(container.querySelector('[data-slot="page-header-icon"]')).toBeNull();
  });

  it("Should omit the count badge when count is undefined", () => {
    const { container } = render(<PageHeader title="Settings" />);
    expect(container.querySelector('[data-slot="page-header-count"]')).toBeNull();
  });
});
