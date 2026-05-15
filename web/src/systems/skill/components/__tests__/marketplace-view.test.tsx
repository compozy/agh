import { UIProvider } from "@agh/ui";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { SkillMarketplaceListingPayload } from "../../types";

import { MarketplaceView } from "../marketplace-view";

function makeListing(
  overrides: Partial<SkillMarketplaceListingPayload> = {}
): SkillMarketplaceListingPayload {
  return {
    name: "test-skill",
    slug: "@test/test-skill",
    author: "test",
    description: "description",
    downloads: 10,
    source: "clawhub",
    version: "1.0.0",
    ...overrides,
  };
}

const LISTINGS: SkillMarketplaceListingPayload[] = [
  makeListing({ name: "alpha", slug: "@compozy/alpha" }),
  makeListing({ name: "beta", slug: "@compozy/beta" }),
  makeListing({ name: "gamma", slug: "@community/gamma" }),
];

function renderView(props: Partial<React.ComponentProps<typeof MarketplaceView>> = {}) {
  const merged: React.ComponentProps<typeof MarketplaceView> = {
    listings: LISTINGS,
    installedSkillNames: new Set<string>(),
    searchQuery: "alpha",
    isSearchEnabled: true,
    isSearching: false,
    searchError: null,
    onSearchChange: vi.fn(),
    onInstall: vi.fn(),
    onUpdate: vi.fn(),
    onRemove: vi.fn(),
    isInstalling: false,
    isUpdating: false,
    isRemoving: false,
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <MarketplaceView {...merged} />
    </UIProvider>
  );
}

describe("MarketplaceView", () => {
  it("Should render the search prompt when no query is entered", () => {
    renderView({ searchQuery: "", isSearchEnabled: false, listings: [] });
    expect(screen.getByTestId("marketplace-search-prompt")).toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-grid")).not.toBeInTheDocument();
  });

  it("Should render the marketplace grid when listings exist", () => {
    renderView();
    const grid = screen.getByTestId("marketplace-grid");
    expect(grid).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-row-alpha")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-row-beta")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-row-gamma")).toBeInTheDocument();
  });

  it("Should call onInstall with the listing slug when Install is clicked", async () => {
    const user = userEvent.setup();
    const onInstall = vi.fn();
    renderView({ onInstall });

    await user.click(screen.getByTestId("install-btn-alpha"));
    expect(onInstall).toHaveBeenCalledWith("@compozy/alpha");
  });

  it("Should disable Install while an install mutation is pending", () => {
    renderView({ isInstalling: true });
    expect(screen.getByTestId("install-btn-alpha")).toBeDisabled();
  });

  it("Should render installed state plus Update and Remove actions for installed listings", () => {
    renderView({ installedSkillNames: new Set(["alpha"]) });
    expect(screen.getByTestId("installed-pill-alpha")).toBeInTheDocument();
    expect(screen.getByTestId("update-btn-alpha")).toBeInTheDocument();
    expect(screen.getByTestId("remove-btn-alpha")).toBeInTheDocument();
    expect(screen.queryByTestId("install-btn-alpha")).not.toBeInTheDocument();
  });

  it("Should call onUpdate with the listing name when Update is clicked", async () => {
    const user = userEvent.setup();
    const onUpdate = vi.fn();
    renderView({ installedSkillNames: new Set(["alpha"]), onUpdate });

    await user.click(screen.getByTestId("update-btn-alpha"));
    expect(onUpdate).toHaveBeenCalledWith("alpha");
  });

  it("Should disable Update while update is pending and show spinner copy", () => {
    renderView({ installedSkillNames: new Set(["alpha"]), isUpdating: true });
    const button = screen.getByTestId("update-btn-alpha");
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent("Updating");
  });

  it("Should confirm before calling onRemove", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    renderView({ installedSkillNames: new Set(["alpha"]), onRemove });

    await user.click(screen.getByTestId("remove-btn-alpha"));
    expect(onRemove).not.toHaveBeenCalled();
    expect(screen.getByTestId("remove-dialog-alpha")).toBeInTheDocument();

    await user.click(screen.getByTestId("confirm-remove-alpha"));
    expect(onRemove).toHaveBeenCalledWith("alpha");
  });

  it("Should not call onRemove when cancel is clicked in the confirm dialog", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    renderView({ installedSkillNames: new Set(["alpha"]), onRemove });

    await user.click(screen.getByTestId("remove-btn-alpha"));
    await user.click(screen.getByTestId("cancel-remove-alpha"));

    expect(onRemove).not.toHaveBeenCalled();
  });

  it("Should disable Remove while removal is pending", () => {
    renderView({ installedSkillNames: new Set(["alpha"]), isRemoving: true });
    expect(screen.getByTestId("remove-btn-alpha")).toBeDisabled();
  });

  it("Should render the loading spinner when searching with no listings yet", () => {
    renderView({ listings: [], isSearching: true });
    expect(screen.getByTestId("marketplace-loading")).toBeInTheDocument();
  });

  it("Should render the empty state when a query returns no listings", () => {
    renderView({ listings: [] });
    expect(screen.getByTestId("marketplace-empty")).toBeInTheDocument();
  });

  it("Should render an inline error message when the search fails", () => {
    renderView({ listings: [], searchError: new Error("clawhub offline") });
    expect(screen.getByTestId("marketplace-error")).toHaveTextContent("clawhub offline");
  });

  it("Should propagate search input changes to onSearchChange", async () => {
    const user = userEvent.setup();
    const onSearchChange = vi.fn();
    renderView({ onSearchChange });

    const input = screen.getByTestId("marketplace-search-input");
    await user.type(input, "x");
    expect(onSearchChange).toHaveBeenCalled();
  });

  it("Should not render the deprecated read-only metadata warning", () => {
    renderView();
    expect(screen.queryByTestId("marketplace-readonly-notice")).not.toBeInTheDocument();
  });
});
