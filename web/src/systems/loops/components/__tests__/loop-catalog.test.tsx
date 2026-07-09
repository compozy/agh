import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", async importOriginal => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    Link: ({ to, params, children, ...props }: Record<string, unknown>) => (
      <a
        href={typeof to === "string" ? to : "#"}
        data-params={JSON.stringify(params)}
        {...(props as Record<string, unknown>)}
      >
        {children as React.ReactNode}
      </a>
    ),
  };
});

const { LoopCatalog } = await import("../catalog/loop-catalog");
const { loopCatalogFixtures } = await import("../../mocks/fixtures");
type LoopCatalogFilter = import("../../lib/loop-catalog").LoopCatalogFilter;
type LoopBindingKind = import("../../lib/loop-bindings").LoopBindingKind;
type LoopCatalogEntry = import("../../types").LoopCatalogEntry;
type ListingViewMode = import("@agh/ui").ListingViewMode;

const BOUND = new Map<string, LoopBindingKind[]>([["software-delivery", ["schedule"]]]);
const DEFAULT_FILTER: LoopCatalogFilter = { kind: "all", category: null, status: null };

function Harness({
  onRun,
  onClearFilters = () => {},
  filter = DEFAULT_FILTER,
  searchQuery = "",
  view = "rows",
}: {
  onRun: (entry: LoopCatalogEntry) => void;
  onClearFilters?: () => void;
  filter?: LoopCatalogFilter;
  searchQuery?: string;
  view?: ListingViewMode;
}) {
  return (
    <LoopCatalog
      boundLoops={BOUND}
      entries={loopCatalogFixtures}
      filter={filter}
      onClearFilters={onClearFilters}
      onRun={onRun}
      searchQuery={searchQuery}
      view={view}
    />
  );
}

describe("LoopCatalog", () => {
  it("Should render grouped rows with success rate and the last-outcome pill", () => {
    render(<Harness onRun={() => {}} />);
    expect(screen.getByTestId("loop-group-read-only")).toBeInTheDocument();
    expect(screen.getByTestId("loop-group-workspace")).toBeInTheDocument();
    expect(screen.getByText("90%")).toBeInTheDocument();
    expect(screen.getByText("100%")).toBeInTheDocument();
    expect(screen.getByText("Running")).toBeInTheDocument();
    expect(screen.getByText("Watching")).toBeInTheDocument();
  });

  it("Should show a binding badge only on rows with an attached loop-target automation", () => {
    render(<Harness onRun={() => {}} />);
    const badges = screen.getAllByTestId("loop-binding-badge");
    expect(badges).toHaveLength(1);
    expect(badges[0].querySelector('[data-binding-kind="schedule"]')).toBeTruthy();
  });

  it("Should filter by kind, hiding the non-matching group", () => {
    render(
      <Harness filter={{ kind: "read-only", category: null, status: null }} onRun={() => {}} />
    );
    expect(screen.queryByTestId("loop-group-workspace")).not.toBeInTheDocument();
    expect(screen.getByTestId("loop-group-read-only")).toBeInTheDocument();
  });

  it("Should filter by category", () => {
    render(<Harness filter={{ kind: "all", category: "watch", status: null }} onRun={() => {}} />);
    expect(screen.getByText("reviews-watch")).toBeInTheDocument();
    expect(screen.queryByText("software-delivery")).not.toBeInTheDocument();
  });

  it("Should filter by search query and offer clear filters", () => {
    const onClearFilters = vi.fn();
    render(<Harness onClearFilters={onClearFilters} onRun={() => {}} searchQuery="zzz-no-match" />);
    expect(screen.getByTestId("loop-catalog-empty")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("loop-catalog-clear-filters"));
    expect(onClearFilters).toHaveBeenCalledTimes(1);
  });

  it("Should render the cards grid when view is cards", () => {
    render(<Harness onRun={() => {}} view="cards" />);
    expect(screen.getByTestId("loop-catalog-card-grid")).toBeInTheDocument();
    expect(screen.getByTestId("loop-catalog-card-software-delivery")).toBeInTheDocument();
    expect(screen.queryByTestId("loop-catalog")).not.toBeInTheDocument();
  });

  it("Should launch a run from the card without navigating to the detail link", () => {
    const onRun = vi.fn();
    render(<Harness onRun={onRun} view="cards" />);
    const card = screen.getByTestId("loop-catalog-card-software-delivery");
    const link = within(card).getByRole("link", { name: "Open software-delivery" });
    const runButton = within(card).getByTestId("loop-catalog-run-software-delivery");
    expect(link).not.toContainElement(runButton);
    fireEvent.click(runButton);
    expect(onRun).toHaveBeenCalledTimes(1);
    expect(onRun.mock.calls[0][0].name).toBe("software-delivery");
  });

  it("Should launch a run inline without navigating to the detail row", () => {
    const onRun = vi.fn();
    render(<Harness onRun={onRun} />);
    const deliveryRow = screen
      .getByText("software-delivery")
      .closest("[data-testid='loop-catalog-row']");
    const runButton = within(deliveryRow as HTMLElement).getByTestId(
      "loop-catalog-run-software-delivery"
    );
    fireEvent.click(runButton);
    expect(onRun).toHaveBeenCalledTimes(1);
    expect(onRun.mock.calls[0][0].name).toBe("software-delivery");
  });

  it("Should keep the inline Run button outside the detail link", () => {
    render(<Harness onRun={() => {}} />);
    const deliveryRow = screen
      .getByText("software-delivery")
      .closest("[data-testid='loop-catalog-row']");
    const row = deliveryRow as HTMLElement;
    const link = within(row).getByRole("link", { name: "Open software-delivery" });
    const runButton = within(row).getByTestId("loop-catalog-run-software-delivery");
    expect(link).not.toContainElement(runButton);
  });
});
