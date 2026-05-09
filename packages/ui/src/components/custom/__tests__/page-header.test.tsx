import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PageHeader } from "../page-header";

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
    expect(slots).toEqual(["page-header-main"]);

    const main = container.querySelector('[data-slot="page-header-main"]');
    const mainSlots = Array.from(main?.children ?? []).map(node => node.getAttribute("data-slot"));
    expect(mainSlots).toEqual(["page-header-title", "page-header-controls", "page-header-meta"]);

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

  it("Should render breadcrumb, subtitle, and status row around the main row", () => {
    const { container } = render(
      <PageHeader
        breadcrumb={<span data-testid="breadcrumb">Settings / Providers</span>}
        title="Providers"
        subtitle={<span data-testid="subtitle">Manage agent provider config.</span>}
        statusRow={<span data-testid="status-row">Daemon online</span>}
      />
    );

    expect(screen.getByTestId("breadcrumb")).toBeInTheDocument();
    expect(screen.getByTestId("subtitle")).toBeInTheDocument();
    expect(screen.getByTestId("status-row")).toBeInTheDocument();

    const header = container.querySelector('[data-slot="page-header"]');
    const slots = Array.from(header?.children ?? []).map(node => node.getAttribute("data-slot"));
    expect(slots).toEqual([
      "page-header-breadcrumb",
      "page-header-main",
      "page-header-subtitle",
      "page-header-status-row",
    ]);
  });
});
